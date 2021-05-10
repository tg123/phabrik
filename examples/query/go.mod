module github.com/tg123/phabrik/examples/query

go 1.16

require (
	github.com/github/certstore v0.1.0
	github.com/tg123/phabrik v0.0.0
)

replace github.com/github/certstore => github.com/tg123/certstore v0.1.1-0.20210416194039-a3d5d6605185

replace github.com/tg123/phabrik => ../../
