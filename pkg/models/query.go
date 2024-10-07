package models

/**
 * Query used with Cribl Search.  Can either use a saved search or run an adhoc query.
 */
type CriblQuery struct {
	Type          string `json:"type"`          // either "adhoc" or "saved"
	Query         string `json:"query"`         // Ad-hoc query (Kusto), when Type is "adhoc"
	SavedSearchId string `json:"savedSearchId"` // ID of the Cribl saved search, when Type is "saved"
}
