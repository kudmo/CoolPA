package core

// ServiceID uniquely identifies a service within the cluster.
// The (Namespace, Name) pair is the service's unique ID.
type ServiceID struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

func (id *ServiceID) ToString() string {
	return id.Namespace + "_" + id.Name
}
