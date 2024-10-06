package utils

import (
	"employee-file-upload/models"
	"io"
	"strconv"

	"github.com/xuri/excelize/v2"
)

func ParseExcel(file io.Reader) ([]models.Employee, error) {
	// Open the Excel file from the provided reader
	xlsx, err := excelize.OpenReader(file)
	if err != nil {
		return nil, err
	}

	// Get the first sheet name
	sheetName := xlsx.GetSheetName(1)
	if sheetName == "" {
		return nil, err
	}

	rows, err := xlsx.GetRows(sheetName)
	if err != nil {
		return nil, err
	}

	var employees []models.Employee
	for i, row := range rows {
		if i == 0 {
			continue
		}

		age, _ := strconv.Atoi(row[4])

		emp := models.Employee{
			FirstName: row[0],
			LastName:  row[1],
			Gender:    row[2],
			Country:   row[3],
			Age:       age,
			Date:      row[5],
		}
		employees = append(employees, emp)
	}

	return employees, nil
}
