CREATE TABLE tidedata
(
    uid serial NOT NULL,
    datetime timestamp,
    date varchar(16),
    day varchar (16),
    time varchar(16),
    predictionft real,
    predictioncm integer,
    highlow varchar (16)
);
