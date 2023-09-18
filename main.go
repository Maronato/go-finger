package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
	"golang.org/x/exp/slog"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

const appName = "finger"

// Version of the application.
var version = "dev"

func main() {
	// Run the server
	if err := Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Allow graceful shutdown
	trapSignalsCrossPlatform(cancel)

	cfg := &Config{}

	// Create a logger and add it to the context
	l := NewLogger(cfg)
	ctx = WithLogger(ctx, l)

	// Create a new root command
	subcommands := []*ff.Command{
		NewServerCmd(cfg),
		NewHealthcheckCmd(cfg),
	}
	cmd := NewRootCmd(cfg, subcommands)

	// Parse and run
	if err := cmd.ParseAndRun(ctx, os.Args[1:], ff.WithEnvVarPrefix("WF")); err != nil {
		if errors.Is(err, ff.ErrHelp) || errors.Is(err, ff.ErrNoExec) {
			fmt.Fprintf(os.Stderr, "\n%s\n", ffhelp.Command(cmd))

			return nil
		}

		return fmt.Errorf("error running command: %w", err)
	}

	return nil
}

func NewServerCmd(cfg *Config) *ff.Command {
	return &ff.Command{
		Name:      "serve",
		Usage:     "serve [flags]",
		ShortHelp: "Start the webfinger server",
		Exec: func(ctx context.Context, args []string) error {
			l := LoggerFromContext(ctx)

			// Parse the webfinger files
			fingermap, err := ParseFingerFile(ctx, cfg)
			if err != nil {
				return fmt.Errorf("error parsing finger files: %w", err)
			}

			l.Info(fmt.Sprintf("Loaded %d webfingers", len(fingermap)))

			// Start the server
			if err := StartServer(ctx, cfg, fingermap); err != nil {
				return fmt.Errorf("error running server: %w", err)
			}

			return nil
		},
	}
}

func NewHealthcheckCmd(cfg *Config) *ff.Command {
	return &ff.Command{
		Name:      "healthcheck",
		Usage:     "healthcheck [flags]",
		ShortHelp: "Check if the server is running",
		Exec: func(ctx context.Context, args []string) error {
			// Create a new client
			client := &http.Client{
				Timeout: 5 * time.Second, //nolint:gomnd // We want to use a constant
			}

			// Create a new request
			reqURL := url.URL{
				Scheme: "http",
				Host:   net.JoinHostPort(cfg.Host, cfg.Port),
				Path:   "/healthz",
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), http.NoBody)
			if err != nil {
				return fmt.Errorf("error creating request: %w", err)
			}

			// Send the request
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("error sending request: %w", err)
			}

			defer resp.Body.Close()

			// Check the response
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("server returned status %d", resp.StatusCode) //nolint:goerr113 // We want to return an error
			}

			return nil
		},
	}
}

type loggerCtxKey struct{}

// NewLogger creates a new logger with the given debug level.
func NewLogger(cfg *Config) *slog.Logger {
	level := slog.LevelInfo
	addSource := false

	if cfg.Debug {
		level = slog.LevelDebug
		addSource = true
	}

	return slog.New(
		slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level:     level,
			AddSource: addSource,
		}),
	)
}

func LoggerFromContext(ctx context.Context) *slog.Logger {
	l, ok := ctx.Value(loggerCtxKey{}).(*slog.Logger)
	if !ok {
		panic("logger not found in context")
	}

	return l
}

func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, l)
}

// https://github.com/caddyserver/caddy/blob/fbb0ecfa322aa7710a3448453fd3ae40f037b8d1/sigtrap.go#L37
// trapSignalsCrossPlatform captures SIGINT or interrupt (depending
// on the OS), which initiates a graceful shutdown. A second SIGINT
// or interrupt will forcefully exit the process immediately.
func trapSignalsCrossPlatform(cancel context.CancelFunc) {
	go func() {
		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, os.Interrupt, syscall.SIGINT)

		for i := 0; true; i++ {
			<-shutdown

			if i > 0 {
				fmt.Printf("\nForce quit\n") //nolint:forbidigo // We want to print to stdout
				os.Exit(1)
			}

			fmt.Printf("\nGracefully shutting down. Press Ctrl+C again to force quit\n") //nolint:forbidigo // We want to print to stdout
			cancel()
		}
	}()
}

type Config struct {
	Debug      bool
	Host       string
	Port       string
	urnPath    string
	fingerPath string
}

// NewRootCmd parses the command line flags and returns a Config struct.
func NewRootCmd(cfg *Config, subcommands []*ff.Command) *ff.Command {
	fs := ff.NewFlagSet(appName)

	for _, cmd := range subcommands {
		cmd.Flags = ff.NewFlagSet(cmd.Name).SetParent(fs)
	}

	cmd := &ff.Command{
		Name:        appName,
		Usage:       fmt.Sprintf("%s <command> [flags]", appName),
		ShortHelp:   fmt.Sprintf("(%s) A webfinger server", version),
		Flags:       fs,
		Subcommands: subcommands,
	}

	// Use 0.0.0.0 as the default host if on docker
	defaultHost := "localhost"
	if os.Getenv("ENV_DOCKER") == "true" {
		defaultHost = "0.0.0.0"
	}

	fs.BoolVar(&cfg.Debug, 'd', "debug", "Enable debug logging")
	fs.StringVar(&cfg.Host, 'h', "host", defaultHost, "Host to listen on")
	fs.StringVar(&cfg.Port, 'p', "port", "8080", "Port to listen on")
	fs.StringVar(&cfg.urnPath, 'u', "urn-file", "urns.yml", "Path to the URNs file")
	fs.StringVar(&cfg.fingerPath, 'f', "finger-file", "fingers.yml", "Path to the fingers file")

	return cmd
}

type Link struct {
	Rel  string `json:"rel"`
	Href string `json:"href,omitempty"`
}

type WebFinger struct {
	Subject    string            `json:"subject"`
	Links      []Link            `json:"links,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

type WebFingerMap map[string]*WebFinger

func ParseFingerFile(ctx context.Context, cfg *Config) (WebFingerMap, error) {
	l := LoggerFromContext(ctx)

	urnMap := make(map[string]string)
	fingerData := make(map[string]map[string]string)

	fingermap := make(WebFingerMap)

	// Read URNs file
	file, err := os.ReadFile(cfg.urnPath)
	if err != nil {
		return nil, fmt.Errorf("error opening URNs file: %w", err)
	}

	if err := yaml.Unmarshal(file, &urnMap); err != nil {
		return nil, fmt.Errorf("error unmarshalling URNs file: %w", err)
	}

	// The URNs file must be a map of strings to valid URLs
	for _, v := range urnMap {
		if _, err := url.Parse(v); err != nil {
			return nil, fmt.Errorf("error parsing URN URIs: %w", err)
		}
	}

	l.Debug("URNs file parsed successfully", slog.Int("number", len(urnMap)), slog.Any("data", urnMap))

	// Read webfingers file
	file, err = os.ReadFile(cfg.fingerPath)
	if err != nil {
		return nil, fmt.Errorf("error opening fingers file: %w", err)
	}

	if err := yaml.Unmarshal(file, &fingerData); err != nil {
		return nil, fmt.Errorf("error unmarshalling fingers file: %w", err)
	}

	l.Debug("Fingers file parsed successfully", slog.Int("number", len(fingerData)), slog.Any("data", fingerData))

	// Parse the webfinger file
	for k, v := range fingerData {
		resource := k

		// Remove leading acct: if present
		if len(k) > 5 && resource[:5] == "acct:" {
			resource = resource[5:]
		}

		// The key must be a URL or email address
		if _, err := mail.ParseAddress(resource); err != nil {
			if _, err := url.Parse(resource); err != nil {
				return nil, fmt.Errorf("error parsing webfinger key (%s): %w", k, err)
			}
		} else {
			// Add acct: back to the key if it is an email address
			resource = fmt.Sprintf("acct:%s", resource)
		}

		// Create a new webfinger
		webfinger := &WebFinger{
			Subject: resource,
		}

		// Parse the fields
		for field, value := range v {
			fieldUrn := field

			// If the key is not already an URN, try to find it in the URNs file
			if _, err := url.Parse(field); err != nil {
				if _, ok := urnMap[field]; ok {
					fieldUrn = urnMap[field]
				}
			}

			// If the value is a valid URI, add it to the links
			if _, err := url.Parse(value); err == nil {
				webfinger.Links = append(webfinger.Links, Link{
					Rel:  fieldUrn,
					Href: value,
				})
			} else {
				// Otherwise add it to the properties
				if webfinger.Properties == nil {
					webfinger.Properties = make(map[string]string)
				}

				webfinger.Properties[fieldUrn] = value
			}
		}

		// Add the webfinger to the map
		fingermap[resource] = webfinger
	}

	l.Debug("Webfinger map built successfully", slog.Int("number", len(fingermap)), slog.Any("data", fingermap))

	return fingermap, nil
}

func WebfingerHandler(_ *Config, webmap WebFingerMap) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := LoggerFromContext(ctx)

		// Only handle GET requests
		if r.Method != http.MethodGet {
			l.Debug("Method not allowed")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

			return
		}

		// Get the query params
		q := r.URL.Query()

		// Get the resource
		resource := q.Get("resource")
		if resource == "" {
			l.Debug("No resource provided")
			http.Error(w, "No resource provided", http.StatusBadRequest)

			return
		}

		// Get and validate resource
		webfinger, ok := webmap[resource]
		if !ok {
			l.Debug("Resource not found")
			http.Error(w, "Resource not found", http.StatusNotFound)

			return
		}

		// Set the content type
		w.Header().Set("Content-Type", "application/jrd+json")

		// Write the response
		if err := json.NewEncoder(w).Encode(webfinger); err != nil {
			l.Debug("Error encoding json")
			http.Error(w, "Error encoding json", http.StatusInternalServerError)

			return
		}

		l.Debug("Webfinger request successful")
	})
}

func HealthCheckHandler(_ *Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

type ResponseWrapper struct {
	http.ResponseWriter

	status int
}

func WrapResponseWriter(w http.ResponseWriter) *ResponseWrapper {
	return &ResponseWrapper{w, 0}
}

func (w *ResponseWrapper) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *ResponseWrapper) Status() int {
	return w.status
}

func (w *ResponseWrapper) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}

	size, err := w.ResponseWriter.Write(b)
	if err != nil {
		return 0, fmt.Errorf("error writing response: %w", err)
	}

	return size, nil
}

func (w *ResponseWrapper) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := LoggerFromContext(ctx)

		start := time.Now()

		// Wrap the response writer
		wrapped := WrapResponseWriter(w)

		// Call the next handler
		next.ServeHTTP(wrapped, r)

		status := wrapped.Status()

		// Log the request
		lg := l.With(
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", status),
			slog.String("remote", r.RemoteAddr),
			slog.Duration("duration", time.Since(start)),
		)

		switch {
		case status >= http.StatusInternalServerError:
			lg.Error("Server error")
		case status >= http.StatusBadRequest:
			lg.Info("Client error")
		default:
			lg.Info("Request completed")
		}
	})
}

const (
	// ReadTimeout is the maximum duration for reading the entire
	// request, including the body.
	ReadTimeout = 5 * time.Second
	// WriteTimeout is the maximum duration before timing out
	// writes of the response.
	WriteTimeout = 10 * time.Second
	// IdleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled.
	IdleTimeout = 30 * time.Second
	// ReadHeaderTimeout is the amount of time allowed to read
	// request headers.
	ReadHeaderTimeout = 2 * time.Second
	// RequestTimeout is the maximum duration for the entire
	// request.
	RequestTimeout = 7 * 24 * time.Hour
)

func StartServer(ctx context.Context, cfg *Config, webmap WebFingerMap) error {
	l := LoggerFromContext(ctx)

	// Create the server mux
	mux := http.NewServeMux()
	mux.Handle("/.well-known/webfinger", WebfingerHandler(cfg, webmap))
	mux.Handle("/healthz", HealthCheckHandler(cfg))

	// Create a new server
	srv := &http.Server{
		Addr: net.JoinHostPort(cfg.Host, cfg.Port),
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
		Handler: LoggingMiddleware(
			RecoveryHandler(
				http.TimeoutHandler(mux, RequestTimeout, "request timed out"),
			),
		),
		ReadHeaderTimeout: ReadHeaderTimeout,
		ReadTimeout:       ReadTimeout,
		WriteTimeout:      WriteTimeout,
		IdleTimeout:       IdleTimeout,
	}

	// Create the errorgroup that will manage the server execution
	eg, egCtx := errgroup.WithContext(ctx)

	// Start the server
	eg.Go(func() error {
		l.Info("Starting server", slog.String("addr", srv.Addr))

		// Use the global context for the server
		srv.BaseContext = func(_ net.Listener) context.Context {
			return egCtx
		}

		return srv.ListenAndServe() //nolint:wrapcheck // We wrap the error in the errgroup
	})
	// Gracefully shutdown the server when the context is done
	eg.Go(func() error {
		// Wait for the context to be done
		<-egCtx.Done()

		l.Info("Shutting down server")
		// Disable the cancel since we don't wan't to force
		// the server to shutdown if the context is canceled.
		noCancelCtx := context.WithoutCancel(egCtx)

		return srv.Shutdown(noCancelCtx) //nolint:wrapcheck // We wrap the error in the errgroup
	})

	srv.RegisterOnShutdown(func() {
		l.Info("Server shutdown complete")
	})

	// Ignore the error if the context was canceled
	if err := eg.Wait(); err != nil && ctx.Err() == nil {
		return fmt.Errorf("server exited with error: %w", err)
	}

	return nil
}

func RecoveryHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := LoggerFromContext(ctx)

		defer func() {
			err := recover()
			if err != nil {
				l.Error("Panic", slog.Any("error", err))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)

				return
			}
		}()

		next.ServeHTTP(w, r)
	})
}
