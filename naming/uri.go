package naming

type UriType int32

const (
	UriTypeAuthorityAbEmpty UriType = iota
	UriTypeAbsolute
	UriTypeRootless
	UriTypeEmpty
)

type UriHostType int32

const (
	UriHostTypeNone UriHostType = iota
	UriHostTypeIPv4
	UriHostTypeIPv6
	UriHostTypeRegName
)

type Uri struct {
	Type         UriType
	Scheme       string
	Authority    string
	HostType     UriHostType
	Host         string
	Port         int32
	Path         string
	Query        string
	Fragment     string
	PathSegments []string
}
