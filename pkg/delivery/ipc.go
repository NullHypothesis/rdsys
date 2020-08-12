package delivery

type Mechanism interface {
	MakeRequest(interface{}, interface{}) error
}
