import requests
import os

BASE_URL = "https://api.mangaupdates.com/v1"

def get_token():
    username = os.getenv("MANGAUPDATES_USERNAME")
    password = os.getenv("MANGAUPDATES_PASSWORD")
    if not username or not password:
        return None
    response = requests.put(f"{BASE_URL}/account/login", json={
        "username": username,
        "password": password
    })
    if response.status_code == 200:
        return response.json().get("context", {}).get("session_token")
    return None

def search_single(title: str, headers: dict):
    response = requests.post(f"{BASE_URL}/series/search", json={
        "search": title,
        "perpage": 5
    }, headers=headers)
    if response.status_code != 200:
        return None, 0, 0
    results = response.json().get("results", [])
    if not results:
        return None, 0, 0
    top = results[0].get("record", {})
    series_id = top.get("series_id")
    score = top.get("bayesian_rating") or 0

    latest_chapter = 0
    if series_id:
        ch_response = requests.get(
            f"{BASE_URL}/series/{series_id}/releases",
            headers=headers
        )
        if ch_response.status_code == 200:
            releases = ch_response.json().get("results", [])
            for release in releases:
                ch_num = release.get("record", {}).get("chapter")
                if ch_num:
                    try:
                        val = float(ch_num)
                        if val > latest_chapter:
                            latest_chapter = val
                    except:
                        pass

    return str(series_id) if series_id else None, round(float(score), 2), latest_chapter

def search_manga(title: str, alt_titles: list = []):
    token = get_token()
    if not token:
        return None, 0, 0
    headers = {"Authorization": f"Bearer {token}"}

    all_titles = [title] + (alt_titles or [])

    for t in all_titles:
        series_id, score, latest = search_single(t, headers)
        if series_id:
            return series_id, score, latest

    return None, 0, 0