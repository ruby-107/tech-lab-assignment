package controllers

import (
	"context"
	"employee-file-upload/database"
	"employee-file-upload/models"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/xuri/excelize/v2"
)

var redisCtx = context.Background()
var redisClient *redis.Client

// Upload Excel and parse

func ImportExcel(c *gin.Context) {
	// Open the Excel file
	f, err := excelize.OpenFile("file_XLS.xlsx")
	if err != nil {
		log.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		log.Fatalf("failed to get rows: %v", err)
	}

	if database.DB == nil {
		log.Fatal("Database connection is not initialized")
	}

	sqlQuery := "INSERT IGNORE INTO employees (id,first_name, last_name, gender, country, age, date) VALUES (?, ?, ?, ?, ?, ?, ?)"

	stmt, err := database.DB.Prepare(sqlQuery)
	if err != nil {
		log.Fatalf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	for i, row := range rows {
		if len(row) == 0 {
			log.Printf("Skipping row %d: row is empty", i)
			continue
		}

		// Check if the row has at least 7 columns
		if len(row) < 7 {
			log.Printf("Skipping row %d: not enough columns (%v)", i, row)
			continue
		}
		id, err := strconv.Atoi(string(row[0]))
		if err != nil {
			log.Printf("Skipping row %d: invalid id (%v)", i, row[0])
			continue

		}

		age, err := strconv.Atoi(string(row[5]))
		if err != nil {
			log.Printf("Skipping row %d: invalid age (%v)", i, row[5])
			continue
		}
		str := fmt.Sprintf("INSERT IGNORE INTO employees (id,first_name, last_name, gender, country, age, date) VALUES (%v,%v,%v,%v,%v,%v,%v)", id, string(row[1]), string(row[2]), string(row[3]), string(row[4]), age, string(row[6]))
		fmt.Println(str)

		_, err = stmt.Exec(id, string(row[1]), string(row[2]), string(row[3]), string(row[4]), age, string(row[6]))
		if err != nil {
			log.Printf("Failed to insert row %d: %v (Values: %v)", i, err, row)
			continue
		}
	}

	fmt.Println("Data inserted successfully!")
}

// Get all employees
func GetEmployees(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection is not initialized"})
		return
	}

	cachedEmployees, err := database.RedisClient.Get(database.RedisCtx, "employees").Result()
	if err == nil {
		var employees []models.Employee
		if err := json.Unmarshal([]byte(cachedEmployees), &employees); err == nil {
			log.Println("Returning cached employees")
			c.JSON(http.StatusOK, employees)
			return
		} else {
			log.Printf("Error unmarshalling cached data: %v", err)
		}
	}

	rows, err := database.DB.Query("SELECT * FROM employees")
	if err != nil {
		log.Printf("Failed to fetch data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch data"})
		return
	}
	defer rows.Close()

	var employees []models.Employee
	for rows.Next() {
		var emp models.Employee
		if err := rows.Scan(&emp.ID, &emp.FirstName, &emp.LastName, &emp.Gender, &emp.Country, &emp.Age, &emp.Date); err != nil {
			log.Printf("Failed to scan data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan data"})
			return
		}
		employees = append(employees, emp)
	}

	if employeesData, err := json.Marshal(employees); err == nil {
		database.RedisClient.Set(database.RedisCtx, "employees", employeesData, 0)
		log.Println("Cached employees in Redis")
	}

	c.JSON(http.StatusOK, employees)
}

// Create employee
func CreateEmployee(c *gin.Context) {
	var emp models.Employee
	if err := c.ShouldBindJSON(&emp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	_, err := database.DB.Exec("INSERT INTO employees (first_name, last_name, gender, country, age, date) VALUES (?, ?, ?, ?, ?, ?)",
		emp.FirstName, emp.LastName, emp.Gender, emp.Country, emp.Age, emp.Date)
	if err != nil {
		log.Printf("Failed to insert employee: %v, Error: %v", emp, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert data"})
		return
	}

	employeesData, err := json.Marshal(emp)
	if err == nil {
		database.RedisClient.Set(database.RedisCtx, "employee:"+strconv.Itoa(emp.ID), employeesData, 0) // Cache employee by ID
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Employee created"})
}

// Update employee

func UpdateEmployee(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var emp models.Employee
	if err := c.ShouldBindJSON(&emp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	_, err = database.DB.Exec("UPDATE employees SET first_name=?, last_name=?, gender=?, country=?, age=?, date=? WHERE id=?",
		emp.FirstName, emp.LastName, emp.Gender, emp.Country, emp.Age, emp.Date, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update data"})
		return
	}

	redisKey := "employee:" + strconv.Itoa(id)
	empJSON, err := json.Marshal(emp)
	if err != nil {
		log.Printf("Error serializing employee data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize data"})
		return
	}
	err = database.RedisClient.Set(redisCtx, redisKey, empJSON, 0).Err()
	if err != nil {
		log.Printf("Could not update employee in Redis: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Employee updated"})
}

func DeleteEmployee(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	log.Printf("Attempting to delete employee with ID: %d", id)

	result, err := database.DB.Exec("DELETE FROM employees WHERE id=?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete data"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "Employee not found"})
		return
	}

	redisKey := "employee:" + strconv.Itoa(id)
	err = database.RedisClient.Del(redisCtx, redisKey).Err()
	if err != nil {
		log.Printf("Could not delete employee from Redis: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Employee deleted"})
}

func GetEmployeeByID(c *gin.Context) {
	id := c.Param("id")

	cachedEmployee, err := database.RedisClient.Get(redisCtx, id).Result()
	if err == nil {
		var emp models.Employee
		json.Unmarshal([]byte(cachedEmployee), &emp)
		c.JSON(http.StatusOK, emp)
		return
	}

	var emp models.Employee
	err = database.DB.QueryRow("SELECT id, first_name, last_name, gender, country, age, date FROM employees WHERE id=?", id).
		Scan(&emp.ID, &emp.FirstName, &emp.LastName, &emp.Gender, &emp.Country, &emp.Age, &emp.Date)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve employee"})
		return
	}

	empJSON, _ := json.Marshal(emp)
	database.RedisClient.Set(redisCtx, id, empJSON, 0).Err()

	c.JSON(http.StatusOK, emp)
}
