# Ski-Verleih

Verleihsystem für Ski, entwickelt mit Spec-Driven Development (SDD): `CLAUDE.md` und diese `README.md` sind die Spezifikation.

**Starten:** `go run .` und http://localhost:8080 öffnen. Tests: `go test ./...`
Daten liegen nur im Speicher und sind nach einem Neustart weg (15 Demo-Ski werden beim Start angelegt).

## Spezifikation

### Entitäten
| Entität | Felder |
|---|---|
| **Ski** (Item) | Nr (eindeutig, z.B. `SKI-001`), Bezeichnung, Typ (Alpin, Carving, Freeride, Langlauf), Preismodell, Basispreis, Höchstpreis |
| **Person** | ID, Name, Geburtstag, Ausweisart (Personalausweis, Reisepass, Führerschein), Ausweisnummer, optionaler Ausweis-Upload (Bild/PDF, max. 2 MB) |
| **Ausleihe** (Rental) | ID, Ski, Person, **Verleihbeginn**, **Verleihdauer (Stunden)**, Preis, Rückgabezeitpunkt |

### Funktionen
1. **Dashboard** mit Kacheln: Verfügbare Ski, Ausgeliehene Ski, Gesamt Ski, Registrierte Personen (zusätzlich: Überfällig, Außer Betrieb, Umsatz) und Schnellzugriff-Buttons (Ski verwalten, Personen verwalten, Neue Ausleihe).
2. **Ski-Verwaltung:** anlegen, auflisten (mit Status verfügbar/ausgeliehen/außer Betrieb), löschen.
3. **Personen:** anlegen mit Ausweisart + Nummer, Ausweis wahlweise **eingeben** (Nr. + Typ) und/oder **hochladen**; auflisten, löschen.
4. **Ausleihe:** Ski und Person wählen, **Beginn** (Datum/Uhrzeit, Standard: jetzt) und **Dauer in Stunden** eingeben. Der Preis wird live angezeigt. Rückgabe per Button.
5. **Banner:** Oben auf der Website wirbt ein Banner für den Gutscheincode `Willkommen10`. Ein Klick auf „Mehr erfahren“ zeigt alle weiteren Gutscheine mit Beschreibungen, die auf ansprechende, auffällige Weise gestaltet sind.
6. **Fußzeile:** Unter jeder Seite stehen „© 2026“ mit dem Namen des Verleihs und die Social-Media-Links (Instagram, Facebook, X, YouTube, TikTok) als kleine Logos. Die Konten existieren nicht; die Links führen nur zur Startseite der jeweiligen Plattform und dienen der Optik.

### Preise (flexibel)
Jeder Ski hat ein eigenes Preismodell:
- **pro Stunde:** Preis = Stunden × Stundenpreis
- **pro Tag:** Preis = angefangene Tage (je 24 h) × Tagespreis

Der Preis ist dynamisch: Es gibt einen Basispreis und einen Höchstpreis.

Wenn mindestens 50 % der Ski gebucht sind, steigt der Preis langsam und dynamisch an (linear). Der Höchstpreis beträgt 150 % des Basispreises. Er soll erreicht werden, wenn alles außer einem Ski gebucht ist. Der Preis darf nie über 150 % steigen.

Wenn jemand zweimal einen Ski bucht, bekommt die Person bei der dritten Buchung 50 % Rabatt auf einen beliebigen Ski. Das gilt bei jeder dritten Buchung. Die rabattierte dritte Buchung gilt dabei als letzte.

Man kann einen Gutscheincode angeben, pro Buchung aber nur einen. Zu Beginn gibt es drei:
- `Willkommen10`: 10 % Rabatt auf die erste Buchung einer Person
- `Birthday10`: 10 % Rabatt am Geburtstag der Person (jede Person hat dafür einen Geburtstag)
- `LOYALTY50`: 50 % Rabatt auf die dritte Buchung

Der Preis wird bei der Ausleihe berechnet und fest gespeichert.

### Regeln
- Ein Ski kann nur einmal gleichzeitig ausgeliehen sein (sonst wird der Fehler „Schon gebucht“ angezeigt).
- Eine Person kann mehrere Ski gleichzeitig buchen.
- Ausgeliehene Ski und Personen mit aktiver Ausleihe können nicht gelöscht werden.
- Dauer mindestens 1 Stunde; Ausleihe nach Ablauf der Dauer ohne Rückgabe gilt als überfällig.
- Ein Ski ist außer Betrieb, wenn bei der Rückgabe ein Schadenbericht hinzugefügt wird. Er wird dann zur Prüfung weitergeleitet. Man kann den Schaden beschreiben oder ein Bild hochladen; beides ist optional. Nach der Prüfung („Prüfung abgeschlossen“) ist der Ski wieder im Betrieb.

### Eingabeprüfung & Sicherheit
Die API vertraut keiner Benutzereingabe und übernimmt Text nicht direkt in den Quellcode des Frontends (Schutz gegen XSS und HTML-Injection).

Beim Upload muss sichergestellt sein, dass die Datei wirklich ein PDF oder Foto ist (und z.B. keine .php-Datei oder etwas anderes, das auf dem Server ausgeführt werden kann). Das beginnt mit der Prüfung der Dateiendung; einfache Umgehungen wie `.pdf.php` dürfen ebenfalls nicht funktionieren.

Es wird kein XML verarbeitet (Schutz gegen XXE-Injection).

### API (JSON)
| Methode | Pfad | Zweck |
|---|---|---|
| GET | `/api/config`, `/api/dashboard` | Beschriftungen, Kennzahlen |
| GET | `/api/vouchers` | Gutscheine mit Beschreibung (Banner, Gutschein-Seite) |
| POST | `/api/quote` | Preis vorab berechnen (inkl. Aufschlag und Gutschein) |
| GET/POST | `/api/items` | Ski auflisten / anlegen |
| DELETE | `/api/items/{nr}` | Ski löschen |
| POST | `/api/items/{nr}/review-done` | Prüfung abgeschlossen (wieder im Betrieb) |
| GET | `/api/items/{nr}/damage-photo` | Schadenfoto |
| GET/POST | `/api/persons` | Personen auflisten / anlegen |
| DELETE | `/api/persons/{id}` | Person löschen |
| GET | `/api/persons/{id}/id-scan` | Ausweis-Upload |
| GET/POST | `/api/rentals` | Ausleihen auflisten / anlegen |
| POST | `/api/rentals/{id}/return` | Rückgabe (optional mit Schadenbericht) |

## Erweiterbarkeit (Teil d)
Der Verleih-Typ ist in `internal/domain/config.go` gebündelt (Name, Typen, Demo-Daten, Gutschein-Texte). Domain, Store, API und Frontend arbeiten mit einem generischen `Item`; dadurch ließ sich das Fahrrad-Projekt (Teil c) leicht aus diesem Projekt ableiten. Eine echte Plugin-Architektur, bei der mehrere Verleih-Typen gleichzeitig laufen (`RentalItem` mit `Ski`, `Bike`, ...), ist nicht umgesetzt.
