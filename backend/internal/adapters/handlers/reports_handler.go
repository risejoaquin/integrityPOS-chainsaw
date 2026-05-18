package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ReportsHandler handles dashboard and analytics endpoints.
type ReportsHandler struct {
	db *sql.DB
}

// NewReportsHandler creates a new ReportsHandler.
func NewReportsHandler(db *sql.DB) *ReportsHandler {
	return &ReportsHandler{db: db}
}

// dashboardData holds today's summary and weekly sales
type dashboardData struct {
	TodaySalesTotal   int64         `json:"today_sales_total"`
	TodayTransactions int64         `json:"today_transactions"`
	AverageTicket     float64       `json:"average_ticket"`
	WeeklySales       []daySale     `json:"weekly_sales"`
	TopProducts       []productSale `json:"top_products"`
}

type daySale struct {
	Date  string `json:"date"`
	Total int64  `json:"total"`
}

type productSale struct {
	ProductID   int64  `json:"product_id"`
	ProductName string `json:"product_name"`
	TotalQty    int64  `json:"total_qty"`
	TotalAmount int64  `json:"total_amount"`
}

// Dashboard returns KPIs, weekly chart data, and top products.
// GET /api/v1/reports/dashboard
func (h *ReportsHandler) Dashboard(c *gin.Context) {
	var data dashboardData

	// Today's totals (no voided)
	h.db.QueryRow(`SELECT COALESCE(SUM(total), 0), COUNT(*) FROM sales WHERE date(created_at) = date('now','localtime') AND voided = 0`).Scan(&data.TodaySalesTotal, &data.TodayTransactions)
	if data.TodayTransactions > 0 {
		data.AverageTicket = float64(data.TodaySalesTotal) / float64(data.TodayTransactions)
	}

	// Weekly sales (last 7 days)
	rows, _ := h.db.Query(`SELECT date(created_at) as d, COALESCE(SUM(total), 0) FROM sales WHERE created_at >= datetime('now','-7 days','localtime') AND voided = 0 GROUP BY date(created_at) ORDER BY d ASC`)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var ws daySale
			if rows.Scan(&ws.Date, &ws.Total) == nil {
				data.WeeklySales = append(data.WeeklySales, ws)
			}
		}
	}

	// Top 5 products
	prodRows, _ := h.db.Query(`SELECT si.product_id, p.name, SUM(si.quantity) as qty, SUM(si.total) as total FROM sale_items si JOIN sales s ON si.sale_id = s.id JOIN products p ON si.product_id = p.id WHERE s.created_at >= datetime('now','-30 days','localtime') AND s.voided = 0 GROUP BY si.product_id ORDER BY qty DESC LIMIT 5`)
	if prodRows != nil {
		defer prodRows.Close()
		for prodRows.Next() {
			var ps productSale
			if prodRows.Scan(&ps.ProductID, &ps.ProductName, &ps.TotalQty, &ps.TotalAmount) == nil {
				data.TopProducts = append(data.TopProducts, ps)
			}
		}
	}

	c.JSON(http.StatusOK, data)
}
