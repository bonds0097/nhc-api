package main

type Status int

const (
	UNKNOWN Status = iota
	UNCONFIRMED
	UNREGISTERED
	REGISTERED
)

var statuses = [...]string{
	"unknown",
	"unconfirmed",
	"unregistered",
	"registered",
}

func (status Status) String() string {
	return statuses[status]
}

type Role int

const (
	USER Role = iota
	ORG_ADMIN
	ORG_SUPER_ADMIN
	GLOBAL_ADMIN
	GLOBAL_SUPER_ADMIN
)

var roles = [...]string{
	"user",
	"org_admin",
	"super_org_admin",
	"global_admin",
	"super_global_admin",
}

func (role Role) String() string {
	return roles[role]
}
