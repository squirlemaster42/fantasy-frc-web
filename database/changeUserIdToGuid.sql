CREATE Extension If Not Exists "uuid-ossp";

-- Add uuid columns and associate to user record row
Alter Table Users Add Column UserUuid uuid;
Update Users Set UserUuid = uuid_generate_v4();
Alter Table UserSessions Add Column UserUuid UUID;
Update UserSessions us Set UserUuid = u.UserUuid From Users u Where us.UserId = u.Id;
Alter Table DraftPlayers Add Column UserUuid UUID;
Update DraftPlayers Set UserUuid = u.UserUuid From Users u Where u.Id = DraftPlayers.Player;
Alter Table DraftReaders Add Column UserUuid UUID;
Update DraftReaders dr Set UserUuid = u.UserUuid From Users u Where u.Id = dr.Player;
Alter Table DraftInvites Add Column InvitedUserUuid UUID;
Alter Table DraftInvites Add Column InvitingUserUuid UUID;
Update DraftInvites di Set InvitedUserUuid = u.UserUuid From Users u Where u.Id = di.InvitedPlayer;
Update DraftInvites di Set InvitingUserUuid = u.UserUuid From Users u Where u.Id = di.InvitingPlayer;
Alter Table Drafts Add Column OwnerUserUuid UUID;
Update Drafts d Set OwnerUserUuid = u.UserUuid From Users u Where u.Id = d.Owner;

-- Drop UserId columns
Alter Table UserSessions Drop UserId;
Alter Table DraftPlayers Drop Player;
Alter Table DraftReaders Drop Player;
Alter Table DraftInvites Drop InvitedPlayer;
Alter Table DraftInvites Drop InvitingPlayer;
Alter Table Drafts Drop Owner;

-- Change PK on Users to GUID
Alter Table Users Drop Constraint Users_pkey;
Alter Table Users Add Primary Key (UserUuid);

-- Update FK contraints
Alter Table UserSessions Add Constraint fk_session_user Foreign Key (UserUuid) References Users (UserUuid);
Alter Table DraftPlayers Add Constraint fk_session_user Foreign Key (UserUuid) References Users (UserUuid);
Alter Table DraftReaders Add Constraint fk_reader_user Foreign Key (UserUuid) References Users (UserUuid);
Alter Table DraftInvites Add Constraint fk_invited_user Foreign Key (InvitedUserUuid) References Users (UserUuid);
Alter Table DraftInvites Add Constraint fk_inviting_user Foreign Key (InvitingUserUuid) References Users (UserUuid);
Alter Table Drafts Add Constraint fk_owner_user Foreign Key (OwnerUserUuid) References Users (UserUuid);

-- Drop Id on Users
Alter Table Users Drop Id;
CREATE Extension If Not Exists "uuid-ossp";

-- Add uuid columns and associate to user record row
Alter Table Users Add Column UserUuid uuid;
Update Users Set UserUuid = uuid_generate_v4();
Alter Table UserSessions Add Column UserUuid UUID;
Update UserSessions us Set UserUuid = u.UserUuid From Users u Where us.UserId = u.Id;
Alter Table DraftPlayers Add Column UserUuid UUID;
Update DraftPlayers Set UserUuid = u.UserUuid From Users u Where u.Id = DraftPlayers.Player;
Alter Table DraftReaders Add Column UserUuid UUID;
Update DraftReaders dr Set UserUuid = u.UserUuid From Users u Where u.Id = dr.Player;
Alter Table DraftInvites Add Column InvitedUserUuid UUID;
Alter Table DraftInvites Add Column InvitingUserUuid UUID;
Update DraftInvites di Set InvitedUserUuid = u.UserUuid From Users u Where u.Id = di.InvitedPlayer;
Update DraftInvites di Set InvitingUserUuid = u.UserUuid From Users u Where u.Id = di.InvitingPlayer;
Alter Table Drafts Add Column OwnerUserUuid UUID;
Update Drafts d Set OwnerUserUuid = u.UserUuid From Users u Where u.Id = d.Owner;

-- Drop UserId columns
Alter Table UserSessions Drop UserId;
Alter Table DraftPlayers Drop Player;
Alter Table DraftReaders Drop Player;
Alter Table DraftInvites Drop InvitedPlayer;
Alter Table DraftInvites Drop InvitingPlayer;
Alter Table Drafts Drop Owner;

-- Change PK on Users to GUID
Alter Table Users Drop Constraint Users_pkey;
Alter Table Users Add Primary Key (UserUuid);

-- Update FK contraints
Alter Table UserSessions Add Constraint fk_session_user Foreign Key (UserUuid) References Users (UserUuid);
Alter Table DraftPlayers Add Constraint fk_session_user Foreign Key (UserUuid) References Users (UserUuid);
Alter Table DraftReaders Add Constraint fk_reader_user Foreign Key (UserUuid) References Users (UserUuid);
Alter Table DraftInvites Add Constraint fk_invited_user Foreign Key (InvitedUserUuid) References Users (UserUuid);
Alter Table DraftInvites Add Constraint fk_inviting_user Foreign Key (InvitingUserUuid) References Users (UserUuid);
Alter Table Drafts Add Constraint fk_owner_user Foreign Key (OwnerUserUuid) References Users (UserUuid);

-- Drop Id on Users
Alter Table Users Drop Id;
