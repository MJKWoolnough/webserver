package main

import (
	"io"
	"os"
	"sort"
	"strings"

	"github.com/MJKWoolnough/gedcom"
)

type Family struct {
	ID            uint
	Husband, Wife *Person
	Children      []*Person
}

type Person struct {
	Gender             byte
	ID                 uint
	FirstName, Surname string
	DOB, DOD           string
	SpouseOf           []*Family
	ChildrenOf         *Family
}

type Index []*Person

func (in Index) Len() int {
	return len(in)
}

func (in Index) Less(i, j int) bool {
	if in[i].Surname == in[j].Surname {
		return in[i].FirstName < in[j].FirstName
	}
	return in[i].Surname < in[j].Surname
}

func (in Index) Swap(i, j int) {
	in[i], in[j] = in[j], in[i]
}

type gedcomData struct {
	People  map[uint]*Person
	Indexes [26]Index
}

var GedcomData gedcomData

func (g gedcomData) Search(terms string) Index {
	in := make(Index, 0, 1024)
	for _, person := range g.People {
		if strings.Contains(person.FirstName+" "+person.Surname, terms) {
			in = append(in, person)
		}
	}
	return in
}

func SetupGedcomDate(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	GedcomData.People = make(map[string]*Person)
	families := make(map[string]*Family)
	ps := make([]*gedcom.Individual, 1024)
	fs := make([]*gedcom.Family, 1024)
	r := gedcom.NewReader(f, gedcom.AllowMissingRequired, gedcom.IgnoreInvalidValue, gedcom.AllowUnknownCharset, gedcom.AllowTerminatorsInValue, gedcom.AllowWrongLength, gedcom.AllowInvalidEscape, gedcom.AllowInvalidChars)
	for {
		record, err := r.Record()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch t := record.(type) {
		case *gedcom.Individual:
			id := idToUint(t.ID)
			GedcomData.People[id] = &Person{ID: id}
			ps = append(ps, t)
		case *gedcom.Family:
			id := idToUint(t.ID)
			familes[id] = &Family{ID: id}
			fs = append(fs, t)
		}
	}
	for _, indi := range ps {
		person := GedcomData.People[idToUint(indi.ID)]
		// set names
		if len(indi.PersonalNameStructure) > 0 {
			name := strings.Split(indi.PersonalNameStructure[0].NamePersonal, "/")
			person.FirstName = name[0]
			if len(name) > 1 {
				person.Surname = name[1]
			}
		}
		if indi.Death.Date != "" {
			person.DOB = string(indi.Birth.Date)
			person.DOD = string(indi.Death.Date)
		}
		switch indi.Gender {
		case "M", "m", "Male", "MALE", "male":
			indi.Gender = 'M'
		case "F", "f", "Female", "FEMALE", "female":
			indi.Gender = 'F'
		default:
			indi.Gender = 'U'
		}
		if len(indi.ChildOf) > 0 {
			person.ChildrenOf = families[idToUint(indi.ChildOf[0].ID)]
		}
		person.SpouseOf = make([]*Family, len(indi.SpouseOf))
		for n, spouse := range indi.SpouseOf {
			person.SpouseOf[n] = family[idToUint(spouse.ID)]
		}
		if len(person.Surname) > 0 {
			n := strings.ToUpper(person.Surname)
			l := n[0]
			if l >= 'A' && l < 'Z' {
				GedcomData.Indexes[l-'A'] = append(GedcomData.Indexes[l-'A'], person)
			}
		}
	}
	for _, fam := range fs {
		family := families[idToUint(fam.ID)]
		family.Husband = GedcomData.People[idToUint(fam.Husband)]
		family.Wife = GedcomData.People[idToUint(fam.Wife)]
		family.Children = make([]*Person, len(fam.Children))
		for n, child := range fam.Children {
			family.Children[n] = GedcomData.People[idToUint(child)]
		}
	}

	for _, index := range GedcomData.Indexes {
		sort.Sort(index)
	}
	return nil
}

func idToUint(id gedcom.Xref) uint {
	var num uint
	for _, n := range id {
		if n >= '0' && n <= '9' {
			num *= 10
			num += n - '0'
		}
	}
	return uint
}
