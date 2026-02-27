package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nununugraha/sains-api/internal/repository"
	"github.com/nununugraha/sains-api/internal/service"
)

// SubscriptionHandler handles subscription and checkout endpoints.
type SubscriptionHandler struct {
	queries         *repository.Queries
	midtransService *service.MidtransService
	emailService    *service.EmailService
	frontendURL     string // for redirect URLs
}

// NewSubscriptionHandler creates a new SubscriptionHandler.
func NewSubscriptionHandler(
	queries *repository.Queries,
	midtransService *service.MidtransService,
	emailService *service.EmailService,
	frontendURL string,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		queries:         queries,
		midtransService: midtransService,
		emailService:    emailService,
		frontendURL:     frontendURL,
	}
}

// ── DTOs ────────────────────────────────────────────────────────────

type subscriptionDTO struct {
	ID            string `json:"id"`
	ProductID     string `json:"product_id"`
	PlanID        string `json:"plan_id"`
	Segment       string `json:"segment"`
	Status        string `json:"status"`
	AmountPaidIDR int    `json:"amount_paid_idr"`
	StartsAt      string `json:"starts_at,omitempty"`
	ExpiresAt     string `json:"expires_at"`
	CreatedAt     string `json:"created_at"`
}

func toSubscriptionDTO(s repository.Subscription) subscriptionDTO {
	dto := subscriptionDTO{
		ID:            uuidToString(s.ID),
		ProductID:     s.ProductID.String,
		PlanID:        uuidToString(s.PlanID),
		Segment:       s.Segment,
		Status:        s.Status.String,
		AmountPaidIDR: int(s.AmountPaidIdr.Int32),
		ExpiresAt:     s.ExpiresAt.Time.Format("2006-01-02T15:04:05Z"),
		CreatedAt:     s.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}
	if s.StartsAt.Valid {
		dto.StartsAt = s.StartsAt.Time.Format("2006-01-02T15:04:05Z")
	}
	return dto
}

// ── Step 2.2: Checkout ──────────────────────────────────────────────

// Checkout handles POST /api/checkout
func (h *SubscriptionHandler) Checkout(c *gin.Context) {
	var req struct {
		PlanID    string `json:"plan_id" binding:"required"`
		UtmSource string `json:"utm_source"` // optional: landing page identifier
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequest(c, "plan_id wajib diisi")
		return
	}

	userIDStr, _ := c.Get("user_id")
	userID := stringToUUID(userIDStr.(string))
	email, _ := c.Get("email")
	userName, _ := c.Get("name")

	// Get plan details
	planUUID := stringToUUID(req.PlanID)
	plan, err := h.queries.GetPricingPlan(c.Request.Context(), planUUID)
	if err != nil {
		RespondNotFound(c, "Plan tidak ditemukan")
		return
	}

	// Check if user already has active subscription for this product
	existing, err := h.queries.GetActiveSubscription(c.Request.Context(),
		repository.GetActiveSubscriptionParams{
			UserID:    userID,
			ProductID: plan.ProductID,
		})
	if err == nil && existing.ID.Valid {
		RespondError(c, http.StatusConflict, ErrCodeConflict,
			"Kamu sudah punya subscription aktif untuk produk ini")
		return
	}

	// Create pending subscription
	sub, err := h.queries.CreateSubscription(c.Request.Context(), repository.CreateSubscriptionParams{
		UserID:        userID,
		ProductID:     plan.ProductID,
		PlanID:        plan.ID,
		Segment:       plan.Segment,
		AmountPaidIdr: pgtype.Int4{Int32: plan.PriceIdr, Valid: true},
		Status:        pgtype.Text{String: "pending", Valid: true},
		ExpiresAt:     pgtype.Timestamptz{}, // will be set when paid
		UtmSource:     pgtype.Text{String: req.UtmSource, Valid: req.UtmSource != ""},
	})
	if err != nil {
		log.Printf("failed to create subscription: %v", err)
		RespondInternalError(c)
		return
	}

	subID := uuidToString(sub.ID)

	// Resolve payer name
	payerName := "Pelanggan"
	if name, ok := userName.(string); ok && name != "" {
		payerName = name
	}

	// Create Midtrans Snap transaction
	snapResp, err := h.midtransService.CreateTransaction(c.Request.Context(), service.CreateTransactionInput{
		OrderID:     subID,
		Amount:      int(plan.PriceIdr),
		PayerEmail:  email.(string),
		PayerName:   payerName,
		Description: fmt.Sprintf("Subscription %s — %s (%s)", plan.ProductID.String, plan.Label.String, plan.Segment),
		FinishURL:   h.frontendURL + "/#/payment/success?sub=" + subID,
	})
	if err != nil {
		log.Printf("midtrans create transaction error: %v", err)
		RespondError(c, http.StatusBadGateway, "PAYMENT_ERROR", "Gagal membuat transaksi pembayaran")
		return
	}

	RespondSuccess(c, http.StatusCreated, gin.H{
		"subscription_id": subID,
		"checkout_url":    snapResp.RedirectURL,
		"snap_token":      snapResp.Token,
		"amount":          plan.PriceIdr,
	}, "Silakan lanjutkan pembayaran")
}

// ── Step 2.3: Midtrans Webhook ──────────────────────────────────────

// MidtransWebhook handles POST /api/midtrans/webhook
func (h *SubscriptionHandler) MidtransWebhook(c *gin.Context) {
	var notif service.MidtransNotification
	if err := c.ShouldBindJSON(&notif); err != nil {
		log.Printf("⚠️ Invalid Midtrans webhook payload: %v", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// Verify signature: SHA512(order_id + status_code + gross_amount + server_key)
	if !h.midtransService.VerifySignature(notif) {
		log.Printf("⚠️ Invalid Midtrans signature for order_id=%s", notif.OrderID)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	log.Printf("📥 Midtrans webhook: order_id=%s status=%s payment_type=%s",
		notif.OrderID, notif.TransactionStatus, notif.PaymentType)

	ctx := c.Request.Context()

	if service.IsPaymentSuccess(notif) {
		// order_id is our subscription UUID
		subID := stringToUUID(notif.OrderID)
		sub, err := h.queries.GetSubscriptionByID(ctx, subID)
		if err != nil {
			log.Printf("❌ Subscription not found for order_id=%s: %v", notif.OrderID, err)
			c.JSON(http.StatusOK, gin.H{"status": "acknowledged"})
			return
		}

		// Activate subscription
		_ = h.queries.UpdateSubscriptionStatus(ctx, repository.UpdateSubscriptionStatusParams{
			ID:     sub.ID,
			Status: pgtype.Text{String: "active", Valid: true},
		})

		// Store Midtrans transaction ID in xendit_invoice_id column (reusing existing column)
		_ = h.queries.SetXenditInvoiceID(ctx, repository.SetXenditInvoiceIDParams{
			ID:              sub.ID,
			XenditInvoiceID: pgtype.Text{String: notif.TransactionID, Valid: true},
		})

		// Activate the user account
		_ = h.queries.SetUserActive(ctx, repository.SetUserActiveParams{
			IsActive: pgtype.Bool{Bool: true, Valid: true},
			ID:       sub.UserID,
		})

		// Send welcome email
		user, userErr := h.queries.GetUserByID(ctx, sub.UserID)
		if userErr == nil {
			expiresStr := sub.ExpiresAt.Time.Format("2 January 2006")
			go func() {
				if emailErr := h.emailService.SendWelcomeEmail(
					ctx, user.Email, user.Name.String,
					sub.ProductID.String, expiresStr,
				); emailErr != nil {
					log.Printf("⚠️ Failed to send welcome email: %v", emailErr)
				}
			}()
		}

		log.Printf("✅ Subscription activated: %s (user=%s, payment=%s)",
			notif.OrderID, uuidToString(sub.UserID), notif.PaymentType)

	} else if service.IsPaymentExpired(notif) {
		subID := stringToUUID(notif.OrderID)
		sub, err := h.queries.GetSubscriptionByID(ctx, subID)
		if err == nil {
			_ = h.queries.UpdateSubscriptionStatus(ctx, repository.UpdateSubscriptionStatusParams{
				ID:     sub.ID,
				Status: pgtype.Text{String: "expired", Valid: true},
			})
			log.Printf("⏰ Subscription expired/cancelled: %s (status=%s)", notif.OrderID, notif.TransactionStatus)
		}
	}

	// Always respond 200 to Midtrans (idempotent)
	c.JSON(http.StatusOK, gin.H{"status": "acknowledged"})
}

// ── Step 2.4: Access Check ──────────────────────────────────────────

// AccessCheck handles GET /api/access-check?product=atomic
func (h *SubscriptionHandler) AccessCheck(c *gin.Context) {
	productID := c.DefaultQuery("product", "atomic")

	userIDStr, _ := c.Get("user_id")
	userID := stringToUUID(userIDStr.(string))

	// Admin always has full access (no subscription needed)
	role, roleExists := c.Get("role")
	if roleExists && role.(string) == "admin" {
		RespondSuccess(c, http.StatusOK, gin.H{
			"granted":    true,
			"product":    productID,
			"expires_at": "2099-12-31T23:59:59Z",
			"segment":    "admin",
		})
		return
	}

	// Check active subscription
	sub, err := h.queries.GetActiveSubscription(c.Request.Context(),
		repository.GetActiveSubscriptionParams{
			UserID:    userID,
			ProductID: pgtype.Text{String: productID, Valid: true},
		})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			RespondError(c, http.StatusForbidden, ErrCodeForbidden,
				"Kamu belum punya akses ke produk ini. Silakan berlangganan.")
			return
		}
		RespondInternalError(c)
		return
	}

	RespondSuccess(c, http.StatusOK, gin.H{
		"granted":    true,
		"product":    productID,
		"expires_at": sub.ExpiresAt.Time.Format("2006-01-02T15:04:05Z"),
		"segment":    sub.Segment,
	})
}

// MySubscriptions handles GET /api/subscriptions/me
func (h *SubscriptionHandler) MySubscriptions(c *gin.Context) {
	userIDStr, _ := c.Get("user_id")
	userID := stringToUUID(userIDStr.(string))

	subs, err := h.queries.ListUserSubscriptions(c.Request.Context(), userID)
	if err != nil {
		RespondInternalError(c)
		return
	}

	result := make([]subscriptionDTO, 0, len(subs))
	for _, s := range subs {
		result = append(result, toSubscriptionDTO(s))
	}

	RespondSuccess(c, http.StatusOK, result)
}

// QuotaStatus handles GET /api/quota-status (public)
func (h *SubscriptionHandler) QuotaStatus(c *gin.Context) {
	ctx := c.Request.Context()

	// Get current counts
	activeSubs, _ := h.queries.CountActiveSubscriptions(ctx)
	activeGuests, _ := h.queries.CountActiveGuestSessions(ctx)

	// Get limits from system_config
	maxSubsStr := "200"
	maxGuestsStr := "50"

	if cfg, err := h.queries.GetConfig(ctx, "max_subscribers"); err == nil {
		maxSubsStr = cfg.Value
	}
	if cfg, err := h.queries.GetConfig(ctx, "max_active_guests"); err == nil {
		maxGuestsStr = cfg.Value
	}

	maxSubs, _ := strconv.Atoi(maxSubsStr)
	maxGuests, _ := strconv.Atoi(maxGuestsStr)

	RespondSuccess(c, http.StatusOK, gin.H{
		"subscribers": gin.H{
			"current": activeSubs,
			"max":     maxSubs,
		},
		"guests": gin.H{
			"current": activeGuests,
			"max":     maxGuests,
		},
	})
}
