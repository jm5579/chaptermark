import requests

ANILIST_URL = "https://graphql.anilist.co"

def search_single(title: str):
    query = '''
    query ($search: String) {
        Media(search: $search, type: MANGA) {
            id
            averageScore
            chapters
        }
    }
    '''
    response = requests.post(ANILIST_URL, json={
        "query": query,
        "variables": {"search": title}
    })
    if response.status_code != 200:
        return None, 0, 0
    data = response.json().get("data", {}).get("Media", {})
    if not data:
        return None, 0, 0
    score = data.get("averageScore") or 0
    chapters = data.get("chapters") or 0
    return data.get("id"), round(float(score) / 10, 2), float(chapters)

def search_manga(title: str, alt_titles: list = []):
    all_titles = [title] + (alt_titles or [])
    for t in all_titles:
        al_id, score, chapters = search_single(t)
        if al_id:
            return al_id, score, chapters
    return None, 0, 0