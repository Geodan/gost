package postgis

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"database/sql"
	gostErrors "github.com/geodan/gost/src/errors"
	"github.com/geodan/gost/src/sensorthings/entities"
)

// GetDatastream retrieves a datastream by id
func (gdb *GostDatabase) GetDatastream(id string) (*entities.Datastream, error) {
	intID, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	sql := fmt.Sprintf("select id, description, unitofmeasurement, public.ST_AsGeoJSON(observedarea) AS observedarea FROM %s.datastream where id = $1", gdb.Schema)
	datastream, err := processDatastream(gdb.Db, sql, intID)
	if err != nil {
		return nil, err
	}

	return datastream, nil
}

// GetDatastreams retrieves all datastreams
func (gdb *GostDatabase) GetDatastreams() ([]*entities.Datastream, error) {
	sql := fmt.Sprintf("select id, description, unitofmeasurement, public.ST_AsGeoJSON(observedarea) AS observedarea FROM %s.datastream", gdb.Schema)
	return processDatastreams(gdb.Db, sql)
}

// GetDatastreamByObservation retrieves a datastream linked to the given observation
func (gdb *GostDatabase) GetDatastreamByObservation(observationID string) (*entities.Datastream, error) {
	tID, err := strconv.Atoi(observationID)
	if err != nil {
		return nil, err
	}

	sql := fmt.Sprintf("select datastream.id, datastream.description, datastream.unitofmeasurement, public.ST_AsGeoJSON(datastream.observedarea) AS observedarea  FROM %s.datastream inner join %s.observation on datastream.id = observation.stream_id where observation.id = $1", gdb.Schema, gdb.Schema)
	return processDatastream(gdb.Db, sql, tID)
}

// GetDatastreamsByThing retrieves all datastreams linked to the given thing
func (gdb *GostDatabase) GetDatastreamsByThing(thingID string) ([]*entities.Datastream, error) {
	tID, err := strconv.Atoi(thingID)
	if err != nil {
		return nil, err
	}

	sql := fmt.Sprintf("select datastream.id, datastream.description, datastream.unitofmeasurement, public.ST_AsGeoJSON(datastream.observedarea) AS observedarea  FROM %s.datastream inner join %s.thing on thing.id = datastream.thing_id where thing.id = $1", gdb.Schema, gdb.Schema)
	return processDatastreams(gdb.Db, sql, tID)
}

// GetDatastreamsBySensor retrieves all datastreams linked to the given sensor
func (gdb *GostDatabase) GetDatastreamsBySensor(sensorID string) ([]*entities.Datastream, error) {
	tID, err := strconv.Atoi(sensorID)
	if err != nil {
		return nil, err
	}

	sql := fmt.Sprintf("select datastream.id, datastream.description, datastream.unitofmeasurement, public.ST_AsGeoJSON(datastream.observedarea) AS observedarea  FROM %s.datastream inner join %s.sensor on sensor.id = datastream.sensor_id where sensor.id = $1", gdb.Schema, gdb.Schema)
	return processDatastreams(gdb.Db, sql, tID)
}

// GetDatastreamsByObservedProperty retrieves all datastreams linked to the given ObservedProerty
func (gdb *GostDatabase) GetDatastreamsByObservedProperty(oID string) ([]*entities.Datastream, error) {
	tID, err := strconv.Atoi(oID)
	if err != nil {
		return nil, err
	}

	sql := fmt.Sprintf("select datastream.id, datastream.description, datastream.unitofmeasurement, public.ST_AsGeoJSON(datastream.observedarea) AS observedarea  FROM %s.datastream inner join %s.observedproperty on observedproperty.id = datastream.observedproperty_id where observedproperty.id = $1", gdb.Schema, gdb.Schema)
	return processDatastreams(gdb.Db, sql, tID)
}

func processDatastream(db *sql.DB, sql string, args ...interface{}) (*entities.Datastream, error) {
	datastreams, err := processDatastreams(db, sql, args...)
	if err != nil {
		return nil, err
	}

	if len(datastreams) == 0 {
		return nil, gostErrors.NewRequestNotFound(errors.New("Datastream not found"))
	}

	return datastreams[0], nil
}

func processDatastreams(db *sql.DB, sql string, args ...interface{}) ([]*entities.Datastream, error) {
	rows, err := db.Query(sql, args...)
	defer rows.Close()

	if err != nil {
		return nil, err
	}

	var datastreams = []*entities.Datastream{}
	for rows.Next() {
		var id int
		var description, unitofmeasurement string
		var observedarea *string

		err := rows.Scan(&id, &description, &unitofmeasurement, &observedarea)
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

		datastream := entities.Datastream{}
		datastream.ID = strconv.Itoa(id)
		datastream.Description = description
		datastream.UnitOfMeasurement = unitOfMeasurementMap
		datastream.ObservedArea = observedAreaMap

		datastreams = append(datastreams, &datastream)
	}

	return datastreams, nil
}

// PostDatastream todo
// TODO: !!!!ADD phenomenonTime SUPPORT!!!!
// TODO: !!!!ADD resulttime SUPPORT!!!!
// TODO: !!!!ADD observationtype SUPPORT!!!!
func (gdb *GostDatabase) PostDatastream(d *entities.Datastream) (*entities.Datastream, error) {
	var dsID int
	tID, err := strconv.Atoi(d.Thing.ID)
	if err != nil || !gdb.ThingExists(tID) {
		return nil, gostErrors.NewBadRequestError(errors.New("Thing does not exist"))
	}

	sID, err := strconv.Atoi(d.Sensor.ID)
	if err != nil || !gdb.SensorExists(sID) {
		return nil, gostErrors.NewBadRequestError(errors.New("Sensor does not exist"))
	}

	oID, err := strconv.Atoi(d.ObservedProperty.ID)
	if err != nil || !gdb.ObservedPropertyExists(oID) {
		return nil, gostErrors.NewBadRequestError(errors.New("ObservedProperty does not exist"))
	}

	unitOfMeasurement, _ := json.Marshal(d.UnitOfMeasurement)
	geom := "NULL"
	if len(d.ObservedArea) != 0 {
		observedAreaBytes, _ := json.Marshal(d.ObservedArea)
		geom = fmt.Sprintf("public.ST_GeomFromGeoJSON('%s')", string(observedAreaBytes[:]))
	}

	sql := fmt.Sprintf("INSERT INTO %s.datastream (description, unitofmeasurement, observedarea, thing_id, sensor_id, observedproperty_id) VALUES ($1, $2, %s, $3, $4, $5) RETURNING id", gdb.Schema, geom)
	err = gdb.Db.QueryRow(sql, d.Description, unitOfMeasurement, tID, sID, oID).Scan(&dsID)
	if err != nil {
		return nil, err
	}

	d.ID = strconv.Itoa(dsID)

	// clear inner entities to serves links upon response
	d.Thing = nil
	d.Sensor = nil
	d.ObservedProperty = nil

	return d, nil
}

// DeleteDatastream tries to delete a Datastream by the given id
func (gdb *GostDatabase) DeleteDatastream(id string) error {
	intID, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	_, err = gdb.Db.Exec(fmt.Sprintf("DELETE FROM %s.datastream WHERE id = $1", gdb.Schema), intID)
	if err != nil {
		return err
	}

	return nil
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
