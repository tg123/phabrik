package federation

type PToPActor int64

const (
	PToPActorDirect PToPActor = iota
	PToPActorFederation
	PToPActorRouting
	PToPActorBroadcast
	PToPActorUpperBound
)

type PToPHeader struct {
	From          NodeInstance
	To            NodeInstance
	Actor         PToPActor
	FromRing      string
	ToRing        string
	ExactInstance bool
}
