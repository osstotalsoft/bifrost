package servicediscovery

//Service is the main component of a service discovery
type Service struct {
	UID       string
	Name      string
	Address   string
	Resource  string
	Secured   bool
	Version   string
	Namespace string
}

//ServiceFunc is an alias
type ServiceFunc func(service Service)
