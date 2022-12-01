package balancer

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"go-balancer/internal/balancer/config"
	"io/fs"
	"net"
	"net/http"
)

// embed the static website frontend files
//
//go:embed static/*
var staticContent embed.FS

type modificationServer struct {
	balancer *balancer

	running bool
	port    int

	Close func() error
}

func NewModificationServer(b *balancer) modificationServer {
	return modificationServer{
		balancer: b,
	}
}

func (m *modificationServer) GetPort() int {
	return m.port
}

func (m *modificationServer) IsRunning() bool {
	return m.running
}

// Runs a http server on a new free port to handle runtime balancer state modification.
// Returns the port the server is listening on, as well as a function that closes the server.
func (m *modificationServer) Start() error {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return err
	}

	m.port = listener.Addr().(*net.TCPAddr).Port
	m.Close = listener.Close

	go m.serve(listener)
	fmt.Printf("Started modification server on port %d\n", m.GetPort())

	go m.startDashboardServer(44444)

	return nil
}

// Starts a http server to serve the static files for a simple web dashboard
func (m *modificationServer) startDashboardServer(port int) {
	staticFs, err := fs.Sub(staticContent, "static")
	if err != nil {
		panic(errors.New("Failed to get static subdir of static files."))
	}
	fs := http.FS(staticFs)

	mux := http.NewServeMux()

	fileServer := http.FileServer(fs)

	// intercept a certain path to give it the port of the modifcation http api
	handleGetBackendPort := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/javascript")
		w.WriteHeader(200)
		w.Write([]byte(fmt.Sprintf("const BACKEND_PORT = %d", m.port)))
	})

	mux.HandleFunc("/javascript/request-url.js", handleGetBackendPort)
	mux.HandleFunc("/", fileServer.ServeHTTP)

	fmt.Printf("Dashboard server listening on port %d\n", port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
	if err != nil {
		fmt.Printf("Dashboard server crashed: %s\n", err.Error())
	}
}

func addCorsHeader(res http.ResponseWriter) {
	headers := res.Header()
	headers.Add("Vary", "Origin")
	headers.Add("Vary", "Access-Control-Request-Method")
	headers.Add("Vary", "Access-Control-Request-Headers")
	headers.Add("Access-Control-Allow-Headers", "Content-Type, Origin, Accept")
	headers.Add("Access-Control-Allow-Methods", "GET, PUT, DELETE, OPTIONS")
}

// Actually start serving connections.
// TODO: make it retry if server crashes
func (m *modificationServer) serve(listener net.Listener) {
	err := http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		switch r.URL.Path {
		case "/backends":
			switch r.Method {
			case "OPTIONS":
				addCorsHeader(w)
				w.WriteHeader(200)
				break
			case "GET":
				m.getBackends(w)
				break
			case "PUT":
				m.putBackend(w, r)
				break
			case "DELETE":
				m.deleteBackend(w, r)
				break
			}
		}
	}))

	fmt.Printf("Modification server crashed: %s\n", err.Error())
}

// Writes the list of backends in json format.
func (m *modificationServer) getBackends(w http.ResponseWriter) {
	// specify its json encoded
	w.Header().Set("Content-Type", "application/json")

	// empty if empty
	if m.balancer.backendManager.GetBackendCount() == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	encoded, err := json.Marshal(m.balancer.backendManager.GetBackends())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)

	// write list to resp
	w.Write(encoded)
}

// Creates a backend from the data in the req
func (m *modificationServer) putBackend(w http.ResponseWriter, r *http.Request) {
	var info config.BackendInfo

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&info)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	err = m.balancer.AddBackends([]config.BackendInfo{info})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Delete a backend from the data in the req
func (m *modificationServer) deleteBackend(w http.ResponseWriter, r *http.Request) {
	var info config.BackendInfo

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&info)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	// doesnt throw errors
	m.balancer.RemoveBackends([]config.BackendInfo{info})

	w.WriteHeader(http.StatusOK)
}
