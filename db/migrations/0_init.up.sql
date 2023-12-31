-- Extensions

-- for uuid auto generation (EDIT NOT NEEDED)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE Users (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	email TEXT NOT NULL UNIQUE,
	username TEXT NOT NULL UNIQUE,
	birthdate DATE NOT NULL,
	password_hash BYTEA NOT NULL,
	created TIMESTAMPTZ NOT NULL,
	last_seen TIMESTAMPTZ NOT NULL
);

CREATE TABLE Sessions (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES Users(id) ON DELETE CASCADE,
	login_time TIMESTAMPTZ NOT NULL,
	last_seen_time TIMESTAMPTZ NOT NULL,
	expires TIMESTAMPTZ NOT NULL,
	user_agent TEXT NOT NULL
);

CREATE TABLE AuthCodes (
	code BYTEA PRIMARY KEY,
	client_id TEXT NOT NULL,
	user_id UUID NOT NULL REFERENCES Users(id) ON DELETE CASCADE,
	scope TEXT NOT NULL,
	expires TIMESTAMPTZ NOT NULL
);

CREATE TABLE AccessTokens (
	token BYTEA PRIMARY KEY,
	auth_code BYTEA NOT NULL,
	refresh_token BYTEA NOT NULL,
	user_id UUID NOT NULL REFERENCES Users(id) ON DELETE CASCADE,
	created TIMESTAMPTZ NOT NULL,
	expires INT NOT NULL
);

CREATE TABLE Podcasts (
	-- REQUIRED TAGS
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	title TEXT NOT NULL,
	description TEXT NOT NULL,
	image_url TEXT NOT NULL,
	language TEXT NOT NULL,
	category INTEGER[] NOT NULL,
	explicit TEXT NOT NULL,
	-- RECOMMENDED TAGS
	author TEXT NOT NULL,
	link_url TEXT NOT NUll,
	owner_name TEXT NOT NUll,
	owner_email TEXT NOT NUll,
	-- SITUATIONAL TAGS
	episodic BOOLEAN DEFAULT TRUE, 
	copyright TEXT NOT NUll,
	block BOOLEAN,
	complete BOOLEAN,
	-- RSS/OTHER
	pub_date TIMESTAMPTZ NOT NULL,
	keywords TEXT NOT NUll,
	summary TEXT NOT NUll,
	rss_url TEXT NOT NULL
);

CREATE TABLE Episodes (
	-- REQUIRED TAGS
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	title TEXT NOT NULL,
	enclosure_url TEXT NOT NULL,
	enclosure_length BIGINT NOT NULL,
	enclosure_type TEXT NOT NULL,
	-- RECOMMENDED TAGS
	pub_date TIMESTAMPTZ NOT NULL,
	description TEXT NOT NULL,
	duration BIGINT NOT NULL,
	link_url TEXT NOT NULL,
	image_url TEXT NOT NULL,
	image_title TEXT NOT NULL,
	explicit TEXT NOT NULL,
	-- SITUATIONAL TAGS
	episode INT NOT NULL,
	season INT NOT NULL,
	episode_type TEXT NOT NULL, -- Full, Trailer, Bonus
	--	block BOOLEAN,
	-- OTHER
	subtitle TEXT NOT NULL,
	summary TEXT NOT NULL,
	encoded TEXT NOT NULL, -- this is the <content:encoded> which sometimes contains show notes
	podcast_id UUID NOT NULL REFERENCES Podcasts(id) ON DELETE CASCADE
);

CREATE TABLE Categories (
	id INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	parent_id INTEGER NOT NULL REFERENCES Categories(id) ON DELETE CASCADE
);

CREATE TABLE Subscriptions (
	user_id UUID REFERENCES Users(id) ON DELETE CASCADE NOT NULL,
	podcast_id UUID REFERENCES Podcasts(id) ON DELETE CASCADE NOT NULL,
	completed_ids UUID[],
	in_progress_ids UUID[],
	PRIMARY KEY(user_id,podcast_id)
);

CREATE TABLE UserEpisodes (
	user_id UUID REFERENCES Users,
	episode_id UUID REFERENCES Episodes,
	offset_millis BIGINT NOT NULL DEFAULT 0,
	last_seen TIMESTAMPTZ NOT NULL,
	played BOOLEAN NOT NULL DEFAULT FALSE,
	PRIMARY KEY(user_id,episode_id)
);

INSERT INTO Categories (id,name,parent_id) VALUES (0, 'nil', 0),
	(1,'Arts', 0),(2,'Books', 1),(3,'Design', 1),(4,'Fashion & Beauty',1),(5,'Food',1),(6,'Performing Arts',1),(7,'Visual Arts',1),
	(8,'Business',0),(9,'Careers',8),(10,'Entrepreneurship',8),(11,'Investing',8),(12,'Management',8),(13,'Marketing',8),(14,'Non-Profit',8),
	(15,'Comedy', 0),(16,'Comedy Interviews',15),(17,'Improv',15),(18,'Stand-Up',15),
	(19,'Education',0),(20,'Courses',19),(21,'How To',19),(22,'Language Learning',19),(23,'Self-Improvement',19),
	(24,'Fiction',0),(25,'Comedy Fiction',24),(26,'Drama',24),(27,'Science Fiction',24),
	(28,'Government',0),
	(29,'History',0),
	(30,'Health & Fitness',0),(31,'Alternative Health',30),(32,'Fitness',30),(33,'Medicine',30),(34,'Mental Health',30),(35,'Nutrition',30),(36,'Sexuality',30),
	(37,'Kids & Family',0),(38,'Education for Kids',37),(39,'Parenting',37),(40,'Pets & Animals',37),
	(41,'Leisure',0),(42,'Animation & Manga',41),(43,'Automotive',41),(44,'Aviation',41),(45,'Crafts',41),(46,'Games',41),(47,'Hobbies',41),(48,'Home & Garden',41),(49,'Video Games',41),
	(50,'Music',0),(51,'Music Commentary',50),(52,'Music History',50),(53,'Music Interviews',50),
	(54,'News',0),(55,'Business News',54),(56,'Daily News',54),(57,'Entertainment News',54),(58,'News Commentary',54),(59,'Politics',54),(60,'Sports News',54),(61,'Tech News',54),
	(62,'Religion & Spirituality',0),(63,'Buddhism',62),(64,'Christianity',62),(65,'Hinduism',62),(66,'Islam',62),(67,'Judaism',62),(68,'Religion',62),(69,'Spirituality',62),
	(70,'Science',0),(71,'Astronomy',70),(72,'Chemistry',70),(73,'Earth Sciences',70),(74,'Life Sciences',70),(75,'Mathematics',70),(76,'Natural Sciences',70),(77,'Nature',70),(78,'Physics',70),(79,'Social Sciences',70),
	(80,'Society & Culture',0),(81,'Documentary',80),(82,'Personal Journals',80),(83,'Philosophy',80),(84,'Places & Travel',80),(85,'Relationships',80),
	(86,'Sports',0),(87,'Baseball',86),(88,'Basketball',86),(89,'Cricket',86),(90,'Fantasy Sports',86),(91,'Football',86),(92,'Golf',86),(93,'Hockey',86),
		(94,'Rugby',86),(95,'Running',86),(96,'Soccer',86),(97,'Swimming',86),(98,'Tennis',86),(99,'Volleyball',86),(100,'Wilderness',86),(101,'Wrestling',86),
	(102,'Technology',0),
	(103,'True Crime',0),
	(104,'TV & Film',0),(105,'After Shows',104),(106,'Film History',104),(107,'Film Interviews',104),(108,'Film Reviews',104),(109,'TV Reviews',104),
	-- added later
	(110,'Stories for Kids',37)
;

-- SEARCH
CREATE TABLE podcasts_search (
	id SERIAL PRIMARY KEY,
	podcast_id UUID NOT NULL REFERENCES Podcasts(id) ON DELETE CASCADE,
	search tsvector
);

CREATE FUNCTION podcasts_search_trigger() RETURNS trigger AS $$
BEGIN
		INSERT INTO podcasts_search(podcast_id,search) VALUES(NEW.id,
			setweight(to_tsvector('english',coalesce(NEW.title,'')), 'A') ||
			setweight(to_tsvector('english',coalesce(NEW.keywords,'')), 'A') ||
			setweight(to_tsvector('english',coalesce(NEW.author,'')), 'B') ||
			setweight(to_tsvector('english',coalesce(NEW.description,'')), 'C') ||
			setweight(to_tsvector('english',coalesce(NEW.summary,'')), 'C')
		);
	return NEW;
END
$$ LANGUAGE plpgsql;

CREATE TRIGGER podcasts_search_trigger AFTER INSERT OR UPDATE ON podcasts
	FOR EACH ROW EXECUTE FUNCTION podcasts_search_trigger();

CREATE INDEX weighted_pod_idx ON podcasts_search USING GIN (search);

-- Create index
CREATE INDEX idx_pub_date ON Podcasts (pub_date);
