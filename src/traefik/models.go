package traefik

// DynamicConfig represents Traefik's dynamic configuration
type DynamicConfig struct {
	HTTP struct {
		Routers     map[string]Router     `json:"routers"`
		Services    map[string]Service    `json:"services"`
		Middlewares map[string]Middleware `json:"middlewares"`
	} `json:"http"`
}

type Router struct {
	EntryPoints []string `json:"entryPoints"`
	Service     string   `json:"service"`
	Rule        string   `json:"rule"`
	Middlewares []string `json:"middlewares,omitempty"`
	TLS         *TLS     `json:"tls,omitempty"`
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

type TLS struct {
	CertResolver string `json:"certResolver,omitempty"`
}

type Middleware struct {
	StripPrefix *StripPrefix `json:"stripPrefix,omitempty"`
	AddPrefix   *AddPrefix   `json:"addPrefix,omitempty"`
	Headers     *Headers     `json:"headers,omitempty"`
	RateLimit   *RateLimit   `json:"rateLimit,omitempty"`
	BasicAuth   *BasicAuth   `json:"basicAuth,omitempty"`
}

type StripPrefix struct {
	Prefixes []string `json:"prefixes"`
}

type AddPrefix struct {
	Prefix string `json:"prefix"`
}

type Headers struct {
	CustomRequestHeaders  map[string]string `json:"customRequestHeaders,omitempty"`
	CustomResponseHeaders map[string]string `json:"customResponseHeaders,omitempty"`
}

type RateLimit struct {
	Average int    `json:"average"`
	Burst   int    `json:"burst"`
	Period  string `json:"period,omitempty"`
}

type BasicAuth struct {
	Users []string `json:"users"`
	Realm string   `json:"realm,omitempty"`
}
