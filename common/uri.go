package common

import (
	"fmt"
)

type UriType int64

const (
	UriTypeAuthorityAbEmpty UriType = iota
	UriTypeAbsolute
	UriTypeRootless
	UriTypeEmpty
)

type UriHostType int64

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

func (u Uri) String() string {
	return fmt.Sprintf(
		"%s:%s",
		u.Scheme, u.Path)
}

// func ParseUri(s string) (uri Uri, err error) {
// 	u, err := url.Parse(s)
// 	if err != nil {
// 		return
// 	}

// 	fmt.Println(u.Host)

// 	uri.Scheme = u.Scheme
// 	uri.Host = u.Hostname()

// 	portstr := u.Port()
// 	if portstr != "" {
// 		var port int64
// 		port, err = strconv.ParseInt(portstr, 10, 32)
// 		if err != nil {
// 			return
// 		}
// 		uri.Port = int32(port)
// 	}

// 	uri.Query = u.RawQuery
// 	uri.Path = u.RawPath
// 	uri.Fragment = u.Fragment

// 	return
// }
