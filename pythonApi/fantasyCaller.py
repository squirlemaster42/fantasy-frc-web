import json
import random
import re
import time
import urllib.parse
from dataclasses import dataclass
from html.parser import HTMLParser
from typing import Optional
from urllib.request import Request, urlopen
from urllib.request import HTTPCookieProcessor, build_opener

TARGET = "https://fantasy-frc.cfh.sh"


@dataclass
class User:
    username: str
    password: str
    client: object
    uuid: str = ""
    is_owner: bool = False
    persona: str = ""


@dataclass
class Draft:
    id: int


def load_user_config(path: str) -> str:
    with open(path, "r") as f:
        return f.read()


def parse_users(user_json: str) -> list[User]:
    users_data = json.loads(user_json)
    return [User(**u) for u in users_data]


def create_user(username: str, password: str, persona: str) -> User:
    jar = HTTPCookieProcessor()
    client = build_opener(jar)
    return User(username=username, password=password, client=client, persona=persona)


def init_users(config_path: str) -> list[User]:
    user_json = load_user_config(config_path)
    users = parse_users(user_json)

    for i, user in enumerate(users):
        users[i] = create_user(user.username, user.password, user.persona)

    populate_auth_toks(users)
    print("Starting to make picks")
    return users


def populate_auth_toks(users: list[User]) -> None:
    for user in users:
        print(f"Making login request for user: {user.username}")
        data = urllib.parse.urlencode({"username": user.username, "password": user.password}).encode()
        req = Request(f"{TARGET}/login", data=data, method="POST")
        req.add_header("Content-Type", "application/x-www-form-urlencoded")

        resp = user.client.open(req)
        if resp.getcode() != 200:
            print(f"Failed to login: {user.username}")
            raise Exception("failed to login")

        print(f"Populate auth token request made: {user.username}, Status: {resp.getcode()}")


def create_random_string(min_len: int, max_len: int) -> str:
    alphabet = "abcdefghijklmnopqrstuvwxyz"
    length = random.randint(min_len, max_len)
    return "".join(random.choice(alphabet) for _ in range(length))


def get_player_uuid(owner: User, draft_id: int, username: str) -> str:
    data = urllib.parse.urlencode({
        "description": "",
        "interval": "0",
        "startTime": "0001-01-01T00:00",
        "endTime": "0001-01-01T00:00",
        "draftName": "",
        "search": username
    }).encode()

    req = Request(f"{TARGET}/u/searchPlayers", data=data, method="POST")
    req.add_header("Content-Type", "application/x-www-form-urlencoded")
    req.add_header("Hx-Current-Url", f"{TARGET}/u/draft/{draft_id}/profile")

    resp = owner.client.open(req)
    if resp.getcode() != 200:
        print(f"Failed to search for username: {username}")
        raise Exception("failed to create draft")

    body = resp.read().decode()
    prefix = '<button hx-target="#inviteTable" hx-swap="outerHTML" name="userUuid" value="'
    
    if body.count(prefix) != 1:
        print(f"Did not find only one user: {username}, Draft Id: {draft_id}")
        raise Exception("err: did not find only one user")

    idx = body.index(prefix) + len(prefix)
    sliced = body[idx:]
    uuid_str = sliced[:sliced.index('"')]

    print(f"Found UUID: {username}, UUID: {uuid_str}")
    return uuid_str


def invite_players_to_draft(owner: User, users: list[User], draft: Draft) -> None:
    for user in users:
        if user.username == owner.username:
            continue

        uuid = get_player_uuid(owner, draft.id, user.username)

        data = urllib.parse.urlencode({
            "description": create_random_string(10, 1000),
            "interval": "0",
            "startTime": "0001-01-01T00:00",
            "endTime": "0001-01-01T00:00",
            "draftName": create_random_string(5, 50),
            "search": "",
            "userUuid": uuid
        }).encode()

        req = Request(f"{TARGET}/u/draft/{draft.id}/invitePlayer", data=data, method="POST")
        req.add_header("Content-Type", "application/x-www-form-urlencoded")

        resp = owner.client.open(req)
        if resp.getcode() != 200:
            print(f"Failed to invite player to draft: {user.username}")
            raise Exception("failed to create draft")

        print(f"Request made: {user.username}, Status: {resp.getcode()}")


def create_draft(user: User) -> Draft:
    print(f"Making request to make draft: {user.username}")

    start_time = time.time() + 60
    start_time_str = time.strftime("%Y-%m-%dT%H:%M", time.localtime(start_time))
    end_time_str = time.strftime("%Y-%m-%dT%H:%M", time.localtime(start_time + 600))

    data = urllib.parse.urlencode({
        "description": create_random_string(10, 1000),
        "interval": "0",
        "startTime": start_time_str,
        "endTime": end_time_str,
        "draftName": create_random_string(5, 50)
    }).encode()

    req = Request(f"{TARGET}/u/createDraft", data=data, method="POST")
    req.add_header("Content-Type", "application/x-www-form-urlencoded")

    resp = user.client.open(req)
    if resp.getcode() != 200:
        print(f"Failed to create draft: {user.username}")
        raise Exception("failed to create draft")

    print(f"Create Draft Request Made: {user.username}, Status: {resp.getcode()}")

    redirect = resp.headers.get("Hx-Redirect", "")
    draft_id_str = redirect.split("/")[3]
    draft_id = int(draft_id_str)

    print(f"Created Draft: Id {draft_id}")
    return Draft(id=draft_id)


def get_invite_id(body: str) -> tuple[int, bool]:
    prefix = '<button hx-target="#pendingTable" hx-swap="outerHTML" name="inviteId" value="'
    if body.count(prefix) == 0:
        return -1, False

    idx = body.index(prefix) + len(prefix)
    sliced = body[idx:]
    end_idx = sliced.index('"')
    id_val = int(sliced[:end_idx])

    return id_val, True


def send_accept_invite(user: User, invite_id: int) -> str:
    data = urllib.parse.urlencode({"inviteId": str(invite_id)}).encode()

    req = Request(f"{TARGET}/u/acceptInvite", data=data, method="POST")
    req.add_header("Content-Type", "application/x-www-form-urlencoded")

    resp = user.client.open(req)
    if resp.getcode() != 200:
        raise Exception("failed to accept invite")

    return resp.read().decode()


def accept_invite(user: User) -> None:
    req = Request(f"{TARGET}/u/viewInvites", method="GET")
    resp = user.client.open(req)

    body = resp.read().decode()

    accept_resp_body = ""
    id_val, found = get_invite_id(body)
    r = 0
    while True:
        if found:
            print(f"Sending accept invite request: {user.username}, Id: {id_val}")
            accept_resp_body = send_accept_invite(user, id_val)
            break
        elif r > 5:
            if user.is_owner:
                break
            print(f"Did not find at least one invite id: {username}")
            raise Exception("error: did not find at least one invite id")
        else:
            r += 1
            time.sleep(0.5)
            id_val, found = get_invite_id(body)

    while found:
        id_val, found = get_invite_id(accept_resp_body)
        if found:
            accept_resp_body = send_accept_invite(user, id_val)


def get_draft_id_page(user: User, draft_id: int) -> str:
    req = Request(f"{TARGET}/u/draft/{draft_id}/profile", method="GET")
    resp = user.client.open(req)
    return resp.read().decode()


def get_current_draft_status(user: User, draft_id: int) -> str:
    profile_page = get_draft_id_page(user, draft_id)
    return parse_draft_status(profile_page)


def parse_draft_status(profile_page: str) -> str:
    prefix = 'id="draftStatus">'
    idx = profile_page.find(prefix)
    if idx == -1:
        return ""
    idx += len(prefix)
    end_idx = profile_page[idx:].find("</div>")
    if end_idx == -1:
        return ""
    return profile_page[idx:idx+end_idx].strip()


def wait_until_draft_state(user: User, draft_id: int, requested_status: str, timeout: int) -> None:
    wait_time = 30
    timeout_time = time.time() + timeout
    current_status = get_current_draft_status(user, draft_id)
    
    while current_status != requested_status:
        print(f"Checking if current draft is in requested status: Requested Status: {requested_status}, Current Status: {current_status}")
        if time.time() > timeout_time:
            raise Exception("wait until draft state timeout reached")

        current_status = get_current_draft_status(user, draft_id)
        time.sleep(wait_time)


def start_draft(user: User, draft_id: int) -> None:
    print(f"Start Draft: Draft Id: {draft_id}, User: {user.username}")
    data = urllib.parse.urlencode({}).encode()

    req = Request(f"{TARGET}/u/draft/{draft_id}/startDraft", data=data, method="POST")
    req.add_header("Content-Type", "application/x-www-form-urlencoded")

    resp = user.client.open(req)
    if resp.getcode() != 200:
        print(f"Failed to start draft: {user.username}")
        raise Exception("failed to create draft")

    print(f"Start Draft Request Made: {user.username}, Status: {resp.getcode()}")
    print(f"Started Draft: {draft_id}")


def init_draft(users: list[User]) -> tuple[User, Draft]:
    owner = random.choice(users)
    owner.is_owner = True
    
    draft = create_draft(owner)
    invite_players_to_draft(owner, users, draft)

    for user in users:
        accept_invite(user)

    start_draft(owner, draft.id)

    current_draft_status = get_current_draft_status(owner, draft.id)
    if get_current_draft_status(owner, draft.id) != "Waiting to Start":
        print(f"Got unexpected draft status: Expected 'Waiting to Start', Actual: {current_draft_status}")
        raise Exception("draft status is not correct")

    wait_until_draft_state(owner, draft.id, "Picking", 300)
    return owner, draft


def is_picking_player(user: User, draft_id: int) -> bool:
    print(f"Getting picking player: Draft Id: {draft_id}, User: {user.username}")

    req = Request(f"{TARGET}/u/draft/{draft_id}/pick", method="GET")
    resp = user.client.open(req)

    if resp.getcode() != 200:
        print(f"Failed to get pick page: Draft Id: {draft_id}, User: {user.username}")
        raise Exception("failed to get pick page")

    print(f"Make pick request make: Draft Id: {draft_id}, User: {user.username}, Status: {resp.getcode()}")

    body = resp.read().decode()
    return 'name="pickInput"' in body


def make_pick_request(draft_id: int, user: User, team: int) -> tuple[bool, str]:
    print(f"Make Pick: Draft Id: {draft_id}, User: {user.username}, Team: {team}")
    data = urllib.parse.urlencode({"pickInput": str(team)}).encode()

    req = Request(f"{TARGET}/u/draft/{draft_id}/makePick", data=data, method="POST")
    req.add_header("Content-Type", "application/x-www-form-urlencoded")

    resp = user.client.open(req)

    if resp.getcode() != 200:
        print(f"Failed to make pick: Team: {team}, Draft Id: {draft_id}, User: {user.username}")
        raise Exception("failed to make pick")

    print(f"Make pick request made: Draft Id: {draft_id}, User: {user.username}, Status: {resp.getcode()}")

    body = resp.read().decode()
    has_error = 'id="pickError"' in body

    if has_error:
        class PickErrorParser(HTMLParser):
            def __init__(self):
                super().__init__()
                self.error_text = ""
                self.in_error = False
            
            def handle_starttag(self, tag, attrs):
                if tag == "div":
                    for attr in attrs:
                        if attr[0] == "id" and attr[1] == "pickError":
                            self.in_error = True

            def handle_endtag(self, tag):
                if tag == "div" and self.in_error:
                    self.in_error = False

            def handle_data(self, data):
                if self.in_error:
                    self.error_text += data

        parser = PickErrorParser()
        parser.feed(body)
        return False, parser.error_text.strip()

    return True, ""


def load_valid_teams() -> list[int]:
    with open("./frc-worlds-2025.csv", "r") as f:
        return [int(line.strip()) for line in f]


def main():
    import sys
    if len(sys.argv) != 2:
        print("Usage: python fantasy_caller.py <config_path>")
        sys.exit(1)
    
    config_path = sys.argv[1]
    users = init_users(config_path)
    owner, draft = init_draft(users)

    valid_teams = load_valid_teams()
    print("Users initialized and draft created successfully")


if __name__ == "__main__":
    main()