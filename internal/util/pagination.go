package util

import (
	"fmt"
	"math"
)

// DefaultPageSize is the default number of items per page.
const DefaultPageSize = 10

// Pagination holds paging state.
type Pagination struct {
	Page     int
	PageSize int
	Total    int64
}

// TotalPages returns the total number of pages.
func (p *Pagination) TotalPages() int {
	if p.Total <= 0 {
		return 1
	}
	return int(math.Ceil(float64(p.Total) / float64(p.PageSize)))
}

// HasPrev returns true if there is a previous page.
func (p *Pagination) HasPrev() bool {
	return p.Page > 1
}

// HasNext returns true if there is a next page.
func (p *Pagination) HasNext() bool {
	return p.Page < p.TotalPages()
}

// PageLabel returns the "第X/Y页" display string.
func (p *Pagination) PageLabel() string {
	return fmt.Sprintf("第%d/%d页", p.Page, p.TotalPages())
}
