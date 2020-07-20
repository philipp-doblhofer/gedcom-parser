package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pquerna/ffjson/ffjson"

	gedcomSpec "github.com/jochenboesmans/gedcom-parser/gedcom"
	"github.com/jochenboesmans/gedcom-parser/model"
	"github.com/jochenboesmans/gedcom-parser/model/child"
	"github.com/jochenboesmans/gedcom-parser/model/family"
	"github.com/jochenboesmans/gedcom-parser/model/individual"
	"github.com/jochenboesmans/gedcom-parser/util"
)

func main() {
	beginTime := time.Now()

	files, err := ioutil.ReadDir("io")
	if err != nil {
		log.Fatal("Unable to read from folder ./io")
	}

	var concurrentlyOpenFiles = make(chan int, 1020)
	waitGroup := &sync.WaitGroup{}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".ged") {
			waitGroup.Add(1)
			go parse(f.Name(), waitGroup, concurrentlyOpenFiles)
		}
	}
	waitGroup.Wait()

	fmt.Printf("total time taken: %f second.\n", float64(time.Since(beginTime))*math.Pow10(-9))
}

func parse(inputFileName string, outerWaitGroup *sync.WaitGroup, concurrentlyOpenFiles chan int) {
	concurrentlyOpenFiles <- 1 // premature increment of semaphore to prevent race condition
	file, err := os.Open("./io/" + inputFileName)
	if err != nil {
		<-concurrentlyOpenFiles
		log.Print(err)
	}

	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)

	recordLines := []*gedcomSpec.Line{}
	waitGroup := &sync.WaitGroup{}

	gedcom := model.ConcurrencySafeGedcom{
		Gedcom: model.Gedcom{},
		Lock:   sync.RWMutex{},
	}

	i := 0
	for fileScanner.Scan() {
		line := ""
		if i == 0 {
			line = strings.TrimPrefix(fileScanner.Text(), "\uFEFF")
		} else {
			line = fileScanner.Text()
		}
		gedcomLine := gedcomSpec.NewLine(&line)

		// interpret record once it's fully read
		if len(recordLines) > 0 && *gedcomLine.Level() == 0 {
			waitGroup.Add(1)
			go interpretRecord(&gedcom, recordLines, waitGroup)
			recordLines = []*gedcomSpec.Line{}
		}
		recordLines = append(recordLines, gedcomLine)
		i++
	}

	waitGroup.Wait()
	err = file.Close()
	if err != nil {
		log.Print(err)
	} else {
		<-concurrentlyOpenFiles
	}

	//if !*useProtobuf {
	gedcomJson, err := ffjson.Marshal(gedcom.Gedcom)
	concurrentlyOpenFiles <- 1
	writeFile, err := os.Create("./io/" + strings.Split(inputFileName, ".")[0] + ".json")
	if err != nil {
		<-concurrentlyOpenFiles
		log.Print(err)
	}
	writer := bufio.NewWriter(writeFile)
	_, err = writer.Write(gedcomJson)
	util.Check(err)
	err = writer.Flush()
	util.Check(err)
	//} else {
	//	// WIP: needs full gedcom protobuf structure to be built
	//	pbPerson := &pb.Person{
	//		Id:        gedcom.Persons[0].Id,
	//		PersonRef: gedcom.Persons[0].PersonRef,
	//		IsLiving:  gedcom.Persons[0].IsLiving,
	//	}
	//
	//	personProto, err := proto.Marshal(pbPerson)
	//	personWriteFile, err := os.Create("./artifacts/personproto")
	//
	//	personWriter := bufio.NewWriter(personWriteFile)
	//	_, err = personWriter.Write(personProto)
	//	util.Check(err)
	//	err = personWriter.Flush()
	//	util.Check(err)
	//}

	err = writeFile.Close()
	if err != nil {
		log.Print(err)
	} else {
		<-concurrentlyOpenFiles
	}
	outerWaitGroup.Done()
}

func interpretRecord(gedcom *model.ConcurrencySafeGedcom, recordLines []*gedcomSpec.Line, waitGroup *sync.WaitGroup) {
	switch *recordLines[0].Tag() {
	case "INDI":
		interpretIndividualRecord(gedcom, recordLines)
	case "FAM":
		interpretFamilyRecord(gedcom, recordLines)
	}
	waitGroup.Done()
}

func interpretIndividualRecord(gedcom *model.ConcurrencySafeGedcom, recordLines []*gedcomSpec.Line) {
	individualXRefID := recordLines[0].XRefID()
	individualInstance := individual.NewIndividual(individualXRefID)
	for i, line := range recordLines {
		if i != 0 && *line.Level() == 0 {
			break
		}
		if *line.Level() == 1 {
			switch *line.Tag() {
			case "NAME":
				name := individual.Name{}
				nameParts := strings.Split(*line.Value(), "/")
				if nameParts[0] != "" || nameParts[1] != "" {
					name.GivenName = nameParts[0]
					name.Surname = nameParts[1]
				} else {
					for _, nameLine := range recordLines[i+1:] {
						if *nameLine.Level() < 2 {
							break
						}
						switch *nameLine.Tag() {
						case "GIVN":
							name.GivenName = *nameLine.Value()
						case "SURN":
							name.Surname = *nameLine.Value()
						}
					}
				}
				if name.GivenName != "" || name.Surname != "" {
					individualInstance.Names = append(individualInstance.Names, &name)
				}
			case "SEX":
				if line.Value() != nil {
					switch *line.Value() {
					case "M":
						individualInstance.Gender = "MALE"
					case "F":
						individualInstance.Gender = "FEMALE"
					}
				}
			}
		}
	}
	gedcom.Lock.Lock()
	gedcom.Individuals = append(gedcom.Individuals, &individualInstance)
	gedcom.Lock.Unlock()
}

func interpretFamilyRecord(gedcom *model.ConcurrencySafeGedcom, recordLines []*gedcomSpec.Line) {
	familyId := recordLines[0].XRefID()
	familyInstance := family.NewFamily(familyId)
	for i, line := range recordLines {
		if i != 0 && *line.Level() == 0 {
			break
		}
		switch *line.Tag() {
		case "HUSB":
			if line.Value() != nil {
				fatherId := line.Value()
				familyInstance.FatherId = fatherId
			}
		case "WIFE":
			if line.Value() != nil {
				motherId := line.Value()
				familyInstance.MotherId = motherId
			}
		case "CHIL":
			if line.Value() != nil {
				childId := line.Value()
				familyInstance.ChildIds = append(familyInstance.ChildIds, childId)
			}
		}

		for _, childId := range familyInstance.ChildIds {
			childInstance := child.NewChild(recordLines[0].XRefID(), childId)
			if familyInstance.MotherId != nil && *familyInstance.MotherId != "" {
				childInstance.RelationshipToMother = true
			}
			if familyInstance.FatherId != nil && *familyInstance.FatherId != "" {
				childInstance.RelationshipToFather = true
			}
			gedcom.Lock.Lock()
			gedcom.Children = append(gedcom.Children, &childInstance)
			gedcom.Lock.Unlock()

		}

		gedcom.Lock.Lock()
		gedcom.Families = append(gedcom.Families, &familyInstance)
		gedcom.Lock.Unlock()
	}
}
