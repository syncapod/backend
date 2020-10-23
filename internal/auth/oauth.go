package auth

// CreateAuthorizationCode creates and saves an authorization code with the client & user id
//func CreateAuthorizationCode(dbClient db.Database, userID *protos.ObjectID, clientID string) (*models.AuthCode, error) {
//	key, err := CreateKey(64)
//	if err != nil {
//		return nil, fmt.Errorf("CreateAuthorizationCode() error creating key: %v", err)
//	}
//	code := models.AuthCode{
//		Code:     key,
//		ClientID: clientID,
//		UserID:   userID,
//		Scope:    models.SubScope,
//	}
//
//	err = dbClient.Insert(database.ColAuthCode, &code)
//	if err != nil {
//		return nil, fmt.Errorf("CreateAuthorizationCode() error inserting auth code: %v", err)
//	}
//
//	return &code, nil
//}
//
//// CreateAccessToken creates and saves an access token with a year of validity
//func CreateAccessToken(dbClient db.Database, authCode *models.AuthCode) (*models.AccessToken, error) {
//	tokenString, err := CreateKey(64)
//	if err != nil {
//		return nil, fmt.Errorf("error creating access token: %v", err)
//	}
//	refreshTokenString, err := CreateKey(64)
//	if err != nil {
//		return nil, fmt.Errorf("error creating access token: %v", err)
//	}
//	token := models.AccessToken{
//		AuthCode:     authCode.Code,
//		Token:        tokenString,
//		RefreshToken: refreshTokenString,
//		UserID:       authCode.UserID,
//		Created:      time.Now(),
//		Expires:      3600,
//	}
//
//	if err := dbClient.Insert(database.ColAccessToken, token); err != nil {
//		return nil, fmt.Errorf("error creating access token: %v", err)
//	}
//
//	return &token, nil
//}
//
//// ValidateAuthCode takes pointer to db client and code string, finds the code and returns it
//func ValidateAuthCode(dbClient db.Database, code string) (*models.AuthCode, error) {
//	var authCode models.AuthCode
//	err := dbClient.FindOne(database.ColAuthCode, &authCode, &db.Filter{"code": code}, nil)
//	if err != nil {
//		return nil, fmt.Errorf("ValidateAuthCode() error finding auth code: %v", err)
//	}
//	if authCode.Code == "" {
//		return nil, errors.New("not found")
//	}
//	return &authCode, nil
//}
//
//// ValidateAccessToken takes pointer to dbclient and token string to lookup and validate AccessToken
//func ValidateAccessToken(dbClient db.Database, token string) (*protos.User, error) {
//	var tokenObj models.AccessToken
//	err := dbClient.FindOne(database.ColAccessToken, &tokenObj, &db.Filter{"token": token}, nil)
//	if err != nil {
//		return nil, err
//	}
//
//	// if expired
//	if tokenObj.Created.Add(time.Second * time.Duration(tokenObj.Expires)).Before(time.Now()) {
//		return nil, errors.New("expired access token")
//	}
//
//	u, err := user.FindUserByID(dbClient, tokenObj.UserID)
//	if err != nil {
//		return nil, err
//	}
//
//	return u, nil
//}
//
//func DeleteOauthAccessToken(dbClient db.Database, token string) error {
//	err := dbClient.Delete(database.ColAccessToken, &db.Filter{"token": token})
//	if err != nil {
//		return fmt.Errorf("error deleting oauth access token: %v", err)
//	}
//	return nil
//}
