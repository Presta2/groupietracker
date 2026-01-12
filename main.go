package main

// Artiste représente les infos qu'on affichera côté front.
type Artiste struct {
	ID              int          `json:"id"`
	Nom             string       `json:"nom"`
	Image           string       `json:"image"`
	Membres         []string     `json:"membres"`
	AnneeCreation   int          `json:"annee_creation"`
	PremierAlbum    string       `json:"premier_album"`
	Genre           string       `json:"genre"` // champ non fourni par l'API, restera vide
	ProchainConcert *Concert     `json:"prochain_concert"`
	DernierConcert  *Concert     `json:"dernier_concert"`
	Setlist         *SetlistInfo `json:"setlist"`
}

// Concert représente une date de concert avec lieu.
type Concert struct {
	Date        string `json:"date"`
	Lieu        string `json:"lieu"`
	EstProchain bool   `json:"est_prochain"`
}

// SetlistInfo représente une setlist simplifiée provenant de l'API Setlist.fm.
type SetlistInfo struct {
	Date      string   `json:"date"`
	Lieu      string   `json:"lieu"`
	Chansons  []string `json:"chansons"`
	SourceURL string   `json:"source_url"`
}

// Réponse brute de l'API (champs en anglais).
type artisteAPI struct {
	ID           int      `json:"id"`
	Nom          string   `json:"name"`
	Image        string   `json:"image"`
	Membres      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	PremierAlbum string   `json:"firstAlbum"`
}
