package main

import "strings"

const (
	SafePermissions     = 0755
	InternalServerError = "internal server error"
	BadRequestError     = "bad request"
)

func ContainsAttachStatement(query string) (string, bool) {
	illegalStatement := false
	statements := strings.Split(query, "\n")

	for _, statement := range statements {

		if strings.HasPrefix(strings.ToUpper(statement), "ATTACH") {
			illegalStatement = true
		}
		if strings.HasPrefix(strings.ToUpper(statement), "DETACH") {
			illegalStatement = true
		}
	}

	return strings.Join(statements, " "), illegalStatement
}
