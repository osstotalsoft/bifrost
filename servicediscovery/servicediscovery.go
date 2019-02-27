package servicediscovery

type Service struct {
	UID       string
	Name      string
	Address   string
	Resource  string
	Secured   bool
	Version   string
	Namespace string
}

type ServiceFunc func(service Service)
