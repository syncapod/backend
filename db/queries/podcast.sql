-- name: InsertPodcast :exec
INSERT INTO Podcasts(id,title,description,image_url,language,category,explicit,author,link_url,owner_name,owner_email,episodic,copyright,block,complete,pub_date,keywords,summary,rss_url)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19);

--
