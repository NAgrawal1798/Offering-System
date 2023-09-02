package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Txn struct
type Transaction struct {
	TxnID            string    `json:"txnId"`
	CustomerID       string    `json:"customerId"`
	Amount           int       `json:"amount"`
	MerchantID       string    `json:"merchantId"`
	MerchantCategory string    `json:"merchantCategory"`
	PostEntryMode    string    `json:"postEntryMode"`
	Timestamp        time.Time `json:"timestamp"`
}

// Offer struct
type Offer struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	RewardType       string          `json:"rewardType"`
	Outcome          float64         `json:"outcome"`
	MinAmount        int             `json:"minAmount"`
	MinMilestone     int             `json:"minMilestone"`
	Details          string          `json:"details"`
	EnabledFor       map[string]bool `json:"enabledFor"`
	MerchantCategory string          `json:"merchantCategory"`
}

// Enable user for the particular offer
func (o *Offer) EnableForUser(userID string) {
	o.EnabledFor[userID] = true
}

// Disable the user for the particular offer
func (o *Offer) DisableForUser(userID string) {
	o.EnabledFor[userID] = false
}

// Map of offer
var offers map[string]*Offer

func enableOfferHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	offerName := vars["offerName"]
	userID := vars["userID"]

	offer, exists := offers[offerName]
	if !exists {
		http.NotFound(w, r)
		return
	}

	offer.EnableForUser(userID)
	fmt.Fprintf(w, "Offer '%s' enabled for user '%s'\n", offerName, userID)
}

func disableOfferHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	offerName := vars["offerName"]
	userID := vars["userID"]

	offer, exists := offers[offerName]
	if !exists {
		http.NotFound(w, r)
		return
	}

	offer.DisableForUser(userID)
	fmt.Fprintf(w, "Offer '%s' disabled for user '%s'\n", offerName, userID)
}

func offersDetailsHandler(w http.ResponseWriter, r *http.Request) {
	allOffers := make([]Offer, 0, len(offers))
	for _, offer := range offers {
		allOffers = append(allOffers, *offer)
	}

	json.NewEncoder(w).Encode(allOffers)
}

func isOfferApplicable(transaction Transaction, offer Offer) bool {
	return transaction.Amount >= offer.MinAmount && transaction.MerchantCategory == offer.MerchantCategory && offer.EnabledFor[transaction.CustomerID]
}

func ApplyBestOfferForTransaction(transaction Transaction, offers map[string]*Offer) (*Offer, error) {
	var bestOffer *Offer
	for _, offer := range offers {
		if isOfferApplicable(transaction, *offer) {
			if bestOffer == nil || offer.Outcome > bestOffer.Outcome {
				bestOffer = offer
			}
		}
	}

	if bestOffer != nil {
		return bestOffer, nil
	}

	return nil, fmt.Errorf("no applicable offer found")
}

func createTransactionHandler(w http.ResponseWriter, r *http.Request) {
	var transaction Transaction
	err := json.NewDecoder(r.Body).Decode(&transaction)
	if err != nil {
		http.Error(w, "Invalid transaction data", http.StatusBadRequest)
		return
	}

	bestOffer, err := ApplyBestOfferForTransaction(transaction, offers)
	if err != nil {
		http.Error(w, "No applicable offer found", http.StatusNotFound)
		return
	}

	fmt.Printf("Applied Offer: %+v\n", *bestOffer)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Transaction processed"))
}

func createOfferHandler(w http.ResponseWriter, r *http.Request) {
	var offer Offer
	err := json.NewDecoder(r.Body).Decode(&offer)
	if err != nil {
		http.Error(w, "Invalid offer data", http.StatusBadRequest)
		return
	}

	offers[offer.ID] = &offer

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Offer created"))
}

func main() {
	offers = make(map[string]*Offer)

	r := mux.NewRouter()
	r.HandleFunc("/enable/{offerName}/{userID}", enableOfferHandler).Methods("POST")
	r.HandleFunc("/disable/{offerName}/{userID}", disableOfferHandler).Methods("POST")
	r.HandleFunc("/offers", offersDetailsHandler).Methods("GET")
	r.HandleFunc("/create-offer", createOfferHandler).Methods("POST")

	r.HandleFunc("/create-transaction", createTransactionHandler).Methods("POST")

	http.Handle("/", r)
	fmt.Println("Server is running on :8080")
	http.ListenAndServe(":8080", nil)
}
