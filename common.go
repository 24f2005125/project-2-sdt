package main

type APIResponse[T any] struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    T      `json:"data"`
	Error   string `json:"error"`
}
