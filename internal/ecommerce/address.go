package ecommerce

// AddressInput is the provider-agnostic intermediate that the per-provider
// Customer/Order mappers populate before handing off to NewAddress.
// Keeps the empty-check + Street assembly logic in one place — each
// upstream API spells the same field a little differently (Province
// vs State vs Region; Address1+Address2 vs []Street; Zip vs Postcode)
// and the per-provider mapper does that translation once.
type AddressInput struct {
	FirstName   string
	LastName    string
	StreetLines []string // joined with "\n"; empty lines are dropped
	City        string
	Region      string // Province / State / Region — providers' choice
	PostCode    string
	Country     string
	Telephone   string
}

// NewAddress returns a pointer to a populated Address, or nil if the
// input is "essentially empty" (no name + no first street line). The nil
// return is load-bearing — the rest of the helpdesk uses nil to render
// "no shipping address" rather than an empty card, so don't switch this
// to returning a zero-value struct.
func NewAddress(in AddressInput) *Address {
	firstStreet := ""
	for _, line := range in.StreetLines {
		if line != "" {
			firstStreet = line
			break
		}
	}
	if in.FirstName == "" && in.LastName == "" && firstStreet == "" {
		return nil
	}
	street := ""
	for _, line := range in.StreetLines {
		if line == "" {
			continue
		}
		if street != "" {
			street += "\n"
		}
		street += line
	}
	return &Address{
		FirstName: in.FirstName,
		LastName:  in.LastName,
		Street:    street,
		City:      in.City,
		Region:    in.Region,
		PostCode:  in.PostCode,
		Country:   in.Country,
		Telephone: in.Telephone,
	}
}
