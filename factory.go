package gioc

type Factory struct {
	Arguments []string
	Create    interface{}
}