package auth

//func TestCreateAuthorizationCode(t *testing.T) {
//	mockDB := mock.CreateDB()
//
//	type args struct {
//		dbClient db.Database
//		userID   *protos.ObjectID
//		clientID string
//	}
//	tests := []struct {
//		name    string
//		args    args
//		wantErr bool
//	}{
//		{
//			name: "valid",
//			args: args{
//				dbClient: mockDB,
//				clientID: "testClient",
//				userID:   protos.ObjectIDFromHex("testUserID"),
//			},
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := CreateAuthorizationCode(tt.args.dbClient, tt.args.userID, tt.args.clientID)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("CreateAuthorizationCode() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !tt.wantErr {
//				var found models.AuthCode
//				err = tt.args.dbClient.FindOne(database.ColAuthCode, &found, &db.Filter{"code": got.Code}, db.CreateOptions())
//				if err != nil {
//					t.Errorf("CreateAuthorizationCode() error looking for auth code = %v", err)
//				}
//			}
//		})
//	}
//}
//
//func TestCreateAccessToken(t *testing.T) {
//	mockDB := mock.CreateDB()
//	authCode := &models.AuthCode{
//		Code:     "secret_code",
//		ClientID: "testClient",
//		UserID:   protos.NewObjectID(),
//		Scope:    models.SubScope,
//	}
//	type args struct {
//		dbClient db.Database
//		authCode *models.AuthCode
//	}
//	tests := []struct {
//		name    string
//		args    args
//		wantErr bool
//	}{
//		{
//			name: "valid",
//			args: args{
//				dbClient: mockDB,
//				authCode: authCode,
//			},
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := CreateAccessToken(tt.args.dbClient, tt.args.authCode)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("CreateAccessToken() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			var found models.AccessToken
//			err = tt.args.dbClient.FindOne(database.ColAccessToken, &found,
//				&db.Filter{"token": got.Token}, db.CreateOptions())
//			if err != nil {
//				t.Errorf("CreateAccessToken() could not find access token: %v", err)
//			}
//		})
//	}
//}
//
//func TestValidateAuthCode(t *testing.T) {
//	mockDB := mock.CreateDB()
//	mockAuthCode, err := CreateAuthorizationCode(mockDB, protos.NewObjectID(), "mockClientID")
//	if err != nil {
//		t.Fatalf("TestValidateAuthCode() failed to set up: %v", err)
//	}
//
//	type args struct {
//		dbClient db.Database
//		code     string
//	}
//	tests := []struct {
//		name    string
//		args    args
//		wantErr bool
//	}{
//		{
//			name: "valid",
//			args: args{
//				dbClient: mockDB,
//				code:     mockAuthCode.Code,
//			},
//			wantErr: false,
//		},
//		{
//			name: "invalid",
//			args: args{
//				dbClient: mockDB,
//				code:     "invalidCode",
//			},
//			wantErr: true,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			_, err := ValidateAuthCode(tt.args.dbClient, tt.args.code)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("ValidateAuthCode() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//		})
//	}
//}
//
//func TestValidateAccessToken(t *testing.T) {
//	mockDB := mock.CreateDB()
//	mockUser := &protos.User{
//		Id:       protos.NewObjectID(),
//		DOB:      ptypes.TimestampNow(),
//		Username: "mockUserID",
//	}
//	insertOrFail(t, mockDB, database.ColUser, mockUser)
//	mockAuthCode, err := CreateAuthorizationCode(mockDB, mockUser.Id, "mockClientID")
//	if err != nil {
//		t.Fatalf("TestValidateAccessToken() error creating mockAuthCode: %v", err)
//	}
//	mockAccessToken, err := CreateAccessToken(mockDB, mockAuthCode)
//	if err != nil {
//		t.Fatalf("TestValidateAccessToken() error creating mockAccessToken: %v", err)
//	}
//
//	type args struct {
//		dbClient db.Database
//		token    string
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    *protos.User
//		wantErr bool
//	}{
//		{
//			name: "valid",
//			args: args{
//				dbClient: mockDB,
//				token:    mockAccessToken.Token,
//			},
//			want:    mockUser,
//			wantErr: false,
//		},
//		{
//			name: "invalid",
//			args: args{
//				dbClient: mockDB,
//				token:    "invalidToken",
//			},
//			want:    nil,
//			wantErr: true,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := ValidateAccessToken(tt.args.dbClient, tt.args.token)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("ValidateAccessToken() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("ValidateAccessToken() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
