CREATE TABLE observations
(
    uid serial NOT NULL,
    datetime timestamp,
    date real,
    day varchar (20),
    time varchar(20),
    predictionft real,
    predictioncm real,
    highlow varchar (20),
);
