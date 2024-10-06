package models

type Employee struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Gender    string `json:"gender"`
	Country   string `json:"country"`
	Age       int    `json:"age"`
	Date      string `json:"date"`
}
