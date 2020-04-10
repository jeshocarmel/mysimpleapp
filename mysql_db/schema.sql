USE simple_db;


# Dump of table USER_TABLE
# ------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `SIMPLE_FORM_DATA` (
  `id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `first_name` varchar(100) NOT NULL,
  `last_name` varchar(100) NOT NULL,
  `country` varchar(100) NOT NULL,
  `favourite_food` varchar(100) NOT NULL,
  `added_date` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
  ) ENGINE=InnoDB DEFAULT CHARSET=latin1;
