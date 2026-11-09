package domain

// Config beschreibt, WAS verliehen wird. Für ein anderes Verleih-Projekt
// (z.B. Fahrräder) werden nur diese Datei, README.md und CLAUDE.md angepasst.
type Config struct {
	AppName      string   `json:"appName"`
	ItemSingular string   `json:"itemSingular"`
	ItemPlural   string   `json:"itemPlural"`
	NrPrefix     string   `json:"nrPrefix"`
	Types        []string `json:"types"`
}

var AppConfig = Config{
	AppName:      "Ski-Verleih",
	ItemSingular: "Ski",
	ItemPlural:   "Ski",
	NrPrefix:     "SKI",
	Types:        []string{"Alpin", "Carving", "Freeride", "Langlauf"},
}

// VoucherText sind die Werbetexte der Gutscheine (Gutschein-Seite und Banner).
type VoucherText struct {
	Emoji       string
	Title       string
	Tagline     string
	Description string
}

var VoucherTexts = map[string]VoucherText{
	CodeWelcome: {"🎿", "Willkommen auf der Piste!", "10 % auf deine allererste Ausleihe",
		"Neu bei uns? Mit diesem Code startest du günstiger in den Schnee. Er gilt einmalig für die erste Buchung jeder Person."},
	CodeBirthday: {"🎂", "Dein Tag, dein Rabatt", "10 % an deinem Geburtstag",
		"Zum Geburtstag schenken wir dir 10 % auf deine Ausleihe. Wir prüfen das Datum automatisch anhand deines Profils."},
	CodeLoyalty: {"🏆", "Treue wird belohnt", "50 % auf jede dritte Buchung",
		"Zwei Ausleihen gemacht? Dann ist die dritte nur halb so teuer. Danach beginnt das Zählen von vorn."},
}

// SeedItems sind Demo-Daten beim Start (In-Memory, gehen beim Neustart verloren).
func SeedItems() []Item {
	type s struct {
		typ, name string
		model     PriceModel
		rate      float64
	}
	list := []s{
		{"Alpin", "Atomic Redster 170", PerHour, 4}, {"Alpin", "Rossignol Experience 168", PerHour, 4},
		{"Alpin", "Völkl Deacon 175", PerHour, 5}, {"Carving", "Head Supershape 165", PerHour, 5},
		{"Carving", "Fischer RC4 160", PerHour, 4.5}, {"Carving", "Elan Amphibio 172", PerHour, 5},
		{"Carving", "Blizzard Quattro 168", PerDay, 30}, {"Freeride", "Salomon QST 180", PerDay, 35},
		{"Freeride", "Black Crows Camox 183", PerDay, 40}, {"Freeride", "Nordica Enforcer 177", PerDay, 38},
		{"Freeride", "Atomic Bent 185", PerDay, 36}, {"Langlauf", "Fischer Classic 190", PerDay, 15},
		{"Langlauf", "Madshus Nano 195", PerDay, 18}, {"Langlauf", "Rossignol X-Tour 188", PerDay, 16},
		{"Alpin", "Dynastar Speed 172", PerHour, 4},
	}
	out := make([]Item, 0, len(list))
	for i, e := range list {
		out = append(out, Item{Nr: fmtNr(AppConfig.NrPrefix, i+1), Type: e.typ, Name: e.name, PriceModel: e.model, Rate: e.rate})
	}
	return out
}

func fmtNr(prefix string, n int) string {
	d := []byte{'0' + byte(n/100%10), '0' + byte(n/10%10), '0' + byte(n%10)}
	return prefix + "-" + string(d)
}
