package go_handlers

import (
    "errors"
    "github.com/bobcatalyst/debug"
    "io/fs"
    "net/http"
    "os"
    "path"
    "path/filepath"
    "runtime"
    "strings"
)

// NewSinglePageAppHandler returns an HTTP handler for serving a single-page application.
//
// In regular builds, the handler serves files from the provided embedFs, which generally should be an embed.FS.
// In debug builds (activated via the -tags debug flag), it attempts to locate and serve static assets from the file system
// using the provided embedPath.
//
// Parameters:
//   - embedFs: an fs.FS providing the embedded static assets.
//   - embedPath: a relative file path to the static assets, used only in debug mode.
//
// Returns:
//   - http.Handler: a handler that serves static files if they exist, or falls back to serving the root file (typically index.html)
//     when a file is not found.
//   - error: an error value if the handler cannot be instantiated in debug mode; nil otherwise.
func NewSinglePageAppHandler(embedFs fs.FS, embedPath string) (http.Handler, error) {
    if debug.Debug {
        return newSinglePageAppHandlerDebug(embedPath)
    }
    return newSinglePageAppHandler(embedFs), nil
}

func newSinglePageAppHandlerDebug(embedPath string) (http.Handler, error) {
    _, staticLocation, _, ok := runtime.Caller(2)
    if !ok {
        return nil, errors.New("could not get caller information")
    }
    staticLocation = filepath.Join(filepath.Dir(staticLocation), embedPath)

    static, err := os.OpenRoot(staticLocation)
    if err != nil {
        return nil, err
    }
    // os.Root.New creates a finalizer that should close os.Root when it is unreachable.
    return newSinglePageAppHandler(static.FS()), nil
}

func newSinglePageAppHandler(fsys fs.FS) http.Handler {
    srv := http.FileServerFS(fsys)
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if canServeFile(fsys, path.Clean(r.URL.Path)) {
            srv.ServeHTTP(w, r)
        } else {
            r := r.Clone(r.Context())
            r.URL.Path = "/"
            srv.ServeHTTP(w, r)
        }
    })
}

func canServeFile(fsys fs.FS, file string) bool {
    f, err := fsys.Open(strings.TrimLeft(file, "/"))
    if err != nil {
        return false
    }
    defer f.Close()

    if stat, err := f.Stat(); err != nil {
        return false
    } else if stat.IsDir() {
        return false
    }
    return true
}
