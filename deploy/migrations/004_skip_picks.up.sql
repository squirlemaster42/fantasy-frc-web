-- Skip picks feature (from optInSkip.sql)
Alter Table DraftPlayers Add Column If Not Exists skipPicks boolean Default False;
