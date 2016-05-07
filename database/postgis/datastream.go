package postgis

import (
	"encoding/json"
	"fmt"
	"strconv"

	"errors"
	gostErrors "github.com/geodan/gost/errors"
	"github.com/geodan/gost/sensorthings/entities"
)

// GetDatastream todo
func (gdb *GostDatabase) GetDatastream(id string) (*entities.Datastream, error) {
	intID, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	var dsID int
	var description, unitofmeasurement string
	var observedarea *string
	sql := fmt.Sprintf("select id, description, unitofmeasurement, ST_AsGeoJSON(observedarea) FROM %s.datastream where id = $1", gdb.Schema)
	err = gdb.Db.QueryRow(sql, intID).Scan(&dsID, &description, &unitofmeasurement, &observedarea)

	if err != nil {
		return nil, gostErrors.NewRequestNotFound(fmt.Errorf("Datastream(%s) does not exist", id))
	}

	unitOfMeasurementMap, err := JSONToMap(&unitofmeasurement)
	if err != nil {
		return nil, err
	}

	observedAreaMap, err := JSONToMap(observedarea)
	if err != nil {
		return nil, err
	}

	datastream := entities.Datastream{
		ID:                strconv.Itoa(dsID),
		Description:       description,
		UnitOfMeasurement: unitOfMeasurementMap,
		ObservedArea:      observedAreaMap,
	}

	return &datastream, nil
}

// GetDatastreams todo
func (gdb *GostDatabase) GetDatastreams() ([]*entities.Datastream, error) {
	sql := fmt.Sprintf("select id, description, unitofmeasurement, ST_AsGeoJSON(observedarea) FROM %s.datastream", gdb.Schema)
	rows, err := gdb.Db.Query(sql)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var datastreams = []*entities.Datastream{}

	for rows.Next() {
		var id int
		var description, unitofmeasurement string
		var observedarea *string

		err = rows.Scan(&id, &description, &unitofmeasurement, &observedarea)
		if err != nil {
			return nil, err
		}

		unitOfMeasurementMap, err := JSONToMap(&unitofmeasurement)
		if err != nil {
			return nil, err
		}

		observedAreaMap, err := JSONToMap(observedarea)
		if err != nil {
			return nil, err
		}

		datastream := entities.Datastream{
			ID:                strconv.Itoa(id),
			Description:       description,
			UnitOfMeasurement: unitOfMeasurementMap,
			ObservedArea:      observedAreaMap,
		}
		datastreams = append(datastreams, &datastream)
	}

	return datastreams, nil
}

// PostDatastream todo
// TODO: !!!!ADD phenomenonTime SUPPORT!!!!
// TODO: !!!!ADD resulttime SUPPORT!!!!
// TODO: !!!!ADD observationtype SUPPORT!!!!
func (gdb *GostDatabase) PostDatastream(d entities.Datastream) (*entities.Datastream, error) {
	var dsID int
	tID, err := strconv.Atoi(d.Thing.ID)
	sID, err2 := strconv.Atoi(d.Sensor.ID)
	oID, err3 := strconv.Atoi(d.ObservedProperty.ID)

	if err != nil || !gdb.ThingExists(tID) {
		return nil, gostErrors.NewBadRequestError(errors.New("Thing does not exist"))
	}

	if err2 != nil || !gdb.SensorExists(sID) {
		return nil, gostErrors.NewBadRequestError(errors.New("Sensor does not exist"))
	}

	if err3 != nil || !gdb.ObservedPropertyExists(oID) {
		return nil, gostErrors.NewBadRequestError(errors.New("ObservedProperty does not exist"))
	}

	unitOfMeasurement, _ := json.Marshal(d.UnitOfMeasurement)
	geom := "NULL"
	if len(d.ObservedArea) != 0 {
		observedAreaBytes, _ := json.Marshal(d.ObservedArea)
		geom = fmt.Sprintf("ST_GeomFromGeoJSON('%s')", string(observedAreaBytes[:]))
	}

	sql := fmt.Sprintf("INSERT INTO %s.datastream (description, unitofmeasurement, observedarea, thing_id, sensor_id, observerproperty_id) VALUES ($1, $2, %s, $3, $4, $5) RETURNING id", gdb.Schema, geom)
	err = gdb.Db.QueryRow(sql, d.Description, unitOfMeasurement, tID, sID, oID).Scan(&dsID)
	if err != nil {
		return nil, err
	}

	d.ID = strconv.Itoa(dsID)

	// clear inner entities to serves links upon response
	d.Thing = nil
	d.Sensor = nil
	d.ObservedProperty = nil

	return &d, nil
}

// DatastreamExists checks if a Datastream is present in the database based on a given id
func (gdb *GostDatabase) DatastreamExists(databaseID int) bool {
	var result bool
	sql := fmt.Sprintf("SELECT exists (SELECT 1 FROM %s.datastream WHERE id = $1 LIMIT 1)", gdb.Schema)
	err := gdb.Db.QueryRow(sql, databaseID).Scan(&result)
	if err != nil {
		return false
	}

	return result
}
