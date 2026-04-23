#!/usr/bin/env python3

import sys
import urllib.request
import json
import urllib.error

def main():
    if len(sys.argv) != 2:
        print("Usage: python fetch_teams_2026.py <TBA_API_KEY>")
        sys.exit(1)

    api_key = sys.argv[1].strip()
    event_key = "2026cmptx"
    url = f"https://www.thebluealliance.com/api/v3/event/{event_key}/teams"

    print(f"Debug: URL = {url}")

    request = urllib.request.Request(url)
    request.add_header("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
    request.add_header("X-TBA-Auth-Key", api_key)
    request.add_header("Accept", "application/json")

    try:
        with urllib.request.urlopen(request) as response:
            print(f"Debug: Response status = {response.status}")
            print(f"Debug: Content-Type = {response.headers.get('Content-Type')}")
            teams = json.load(response)
    except urllib.error.HTTPError as e:
        print(f"HTTP Error {e.code}: {e.reason}")
        body = e.read().decode()
        print(f"Response body: {body}")
        sys.exit(1)
    except Exception as e:
        print(f"Error fetching teams: {e}")
        sys.exit(1)

    team_numbers = []
    for team in teams:
        if "team_number" in team:
            team_numbers.append(str(team["team_number"]))

    team_numbers.sort(key=lambda x: int(x))

    with open("frc-worlds-2026.csv", "w") as f:
        for num in team_numbers:
            f.write(num + "\n")

    print(f"Successfully wrote {len(team_numbers)} teams to frc-worlds-2026.csv")

if __name__ == "__main__":
    main()
