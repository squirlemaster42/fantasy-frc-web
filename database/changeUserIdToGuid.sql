CREATE Extension "uuid_ossp";

-- Alter columns and update data
Alter Table Users Add Column UserGuid;
Update Users Set UserGuid = uuid_generate_v4();
Alter Table UserSessions Add Column UserGuid UUID;
Update UserSessions us Set UserGuid = u.UserGuid From Users u Where us.UserId = u.Id;
Alter Table DraftPlayers Add Column UserGuid UUID;
Update DraftPlayers Set UserGuid = u.UserGuid From Users u Where u.Id = DraftPlayers.Player;
Alter Table DraftReaders Add Column UserGuid UUID;
Update DraftReaders dr Set UserGuid = u.UserGuid From Users u Where u.Id = dr.Player;
Alter Table DraftInvites Add Column InvitedUserGuid UUID;
Alter Table DraftInvites Add Column InvitingUserGuid UUID;
Update DraftInvites di Set InvitedUserGuid = u.UserGuid From Users u Where u.Id = di.InvitedPlayer;
Update DraftInvites di Set InvitingUserGuid = u.UserGuid From Users u Where u.Id = di.InvitingPlayer;
Alter Table Drafts Add Column OwnerUserGuid UUID;
Update Drafts d Set OwnerUserGuid = u.UserGuid From Users u Where u.Id = d.Owner;

-- Drop UserId columns
Alter Table UserSessions Drop UserId;
Alter Table DraftPlayers Drop Player;
Alter Table DraftReaders Drop Player;
Alter Table DraftInvites Drop InvitedPlayer;
Alter Table DraftInvites Drop InvitingPlayer;
Alter Table Drafts Drop Owner;

-- Change PK on Users to GUID
Alter Table Users Drop Constraint Users_pkey;
Alter Table Users Add Primary Key (UserGuid);

-- Update FK contraints
Alter Table UserSessions Add Constraint fk_session_user Foreign Key (UserGuid) References Users (UserGuid);
Alter Table DraftPlayers Add Constraint fk_session_user Foreign Key (UserGuid) References Users (UserGuid);
Alter Table DraftReaders Add Constraint fk_reader_user Foreign Key (UserGuid) References Users (UserGuid);
Alter Table DraftInvites Add Constraint fk_invited_user Foreign Key (InvitedUserGuid) References Users (UserGuid);
Alter Table DraftInvites Add Constraint fk_inviting_user Foreign Key (InvitingUserGuid) References Users (UserGuid);
Alter Table Drafts Add Constraint fk_owner_user Foreign Key (OwnerUserGuid) References Users (UserGuid);

-- Drop Id on Users
Alter Table Users Drop Id;
