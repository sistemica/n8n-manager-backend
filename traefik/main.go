package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type DynamicConfig struct {
	HTTP struct {
		Routers    map[string]Router     `json:"routers"`
		Services   map[string]Service    `json:"services"`
		Middleware map[string]Middleware `json:"middlewares"`
	} `json:"http"`
}

type Router struct {
	EntryPoints []string `json:"entryPoints"`
	Service     string   `json:"service"`
	Rule        string   `json:"rule"`
	Middleware  []string `json:"middlewares,omitempty"`
}

type Service struct {
	LoadBalancer *LoadBalancer `json:"loadBalancer"`
}

type LoadBalancer struct {
	Servers []Server `json:"servers"`
}

type Server struct {
	URL string `json:"url"`
}

type Middleware struct {
	StripPrefix *StripPrefix `json:"stripPrefix,omitempty"`
}

type StripPrefix struct {
	Prefixes []string `json:"prefixes"`
}

func main() {
	http.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		config := DynamicConfig{}
		config.HTTP.Routers = make(map[string]Router)
		config.HTTP.Services = make(map[string]Service)
		config.HTTP.Middleware = make(map[string]Middleware)

		// Configure httpbin router
		config.HTTP.Routers["httpbin"] = Router{
			EntryPoints: []string{"web"},
			Service:     "httpbin-service",
			Rule:        "PathPrefix(`/httpbin`)",
			Middleware:  []string{"httpbin-strip"},
		}

		// Configure httpbin service pointing to our Docker container
		config.HTTP.Services["httpbin-service"] = Service{
			LoadBalancer: &LoadBalancer{
				Servers: []Server{
					{URL: "http://httpbin:80"},
				},
			},
		}

		// Configure strip prefix middleware
		config.HTTP.Middleware["httpbin-strip"] = Middleware{
			StripPrefix: &StripPrefix{
				Prefixes: []string{"/httpbin"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))

		if err := json.NewEncoder(w).Encode(config); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	log.Println("Starting config server on :9000")
	if err := http.ListenAndServe(":9000", nil); err != nil {
		log.Fatal(err)
	}
}
