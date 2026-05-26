package services

import (
	"math"
	"strconv"
)

type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	OffSet     int    `json:"-"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

func NewPagination(pageStr, limitStr string) (*Pagination, error) {
	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	return &Pagination{
		Page:   page,
		Limit:  limit,
		OffSet: (page - 1) * limit,
	}, nil
}

func (p *Pagination) Calculate() {
	if p.Total > 0 && p.Limit > 0 {
		p.TotalPages = int(math.Ceil(float64(p.Total) / float64(p.Limit)))
	}
}