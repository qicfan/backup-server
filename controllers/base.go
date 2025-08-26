package controllers

type APIResponseCode int

const (
	Success APIResponseCode = iota
	BadRequest
)

type APIResponse[T any] struct {
	Code    APIResponseCode `json:"code"`
	Message string          `json:"message"`
	Data    T               `json:"data"`
}
