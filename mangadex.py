import requests

BASE_URL = "https://api.mangadex.org"

def search_manga(title: str):
    response = requests.get(f"{BASE_URL}/manga", params={
        "title": title,
        "limit": 10,
        "contentRating[]": ["safe", "suggestive", "erotica"],
        "includes[]": ["cover_art", "author"]
    })
    if response.status_code != 200:
        return []
    results = []
    for item in response.json().get("data", []):
        attributes = item.get("attributes", {})
        title_data = attributes.get("title", {})
        title_str = title_data.get("en") or list(title_data.values())[0] if title_data else "Unknown"

        alt_titles_raw = attributes.get("altTitles", [])
        alt_titles = []
        for alt in alt_titles_raw:
            for lang, val in alt.items():
                if val and val not in alt_titles:
                    alt_titles.append(val)

        cover_url = None
        author = None
        for rel in item.get("relationships", []):
            if rel["type"] == "cover_art":
                filename = rel.get("attributes", {}).get("fileName")
                if filename:
                    cover_url = f"https://uploads.mangadex.org/covers/{item['id']}/{filename}.256.jpg"
            if rel["type"] == "author":
                author = rel.get("attributes", {}).get("name")

        score = attributes.get("rating", {}).get("average") or 0
        results.append({
            "mangadex_id": item["id"],
            "title": title_str,
            "alt_titles": alt_titles,
            "cover_url": cover_url,
            "description": (attributes.get("description") or {}).get("en", "No description available."),
            "status": attributes.get("status", "Unknown"),
            "genres": ", ".join([
                tag["attributes"]["name"].get("en", "")
                for tag in attributes.get("tags", [])
                if tag["attributes"]["group"] == "genre"
            ]),
            "author": author or "Unknown",
            "mangadex_score": round(float(score), 2) if score else 0,
        })
    return results

def get_latest_chapter(mangadex_id: str):
    try:
        highest = 0
        offset = 0
        limit = 500

        while True:
            response = requests.get(f"{BASE_URL}/manga/{mangadex_id}/feed", params={
                "limit": limit,
                "offset": offset,
                "contentRating[]": ["safe", "suggestive", "erotica", "pornographic"],
                "order[chapter]": "desc"
            }, timeout=10)

            if response.status_code != 200:
                break

            data = response.json()
            chapters = data.get("data", [])

            if not chapters:
                break

            for chapter in chapters:
                ch_num = chapter.get("attributes", {}).get("chapter")
                if ch_num:
                    try:
                        val = float(ch_num)
                        if val > highest:
                            highest = val
                    except:
                        pass

            total = data.get("total", 0)
            offset += limit
            if offset >= total:
                break

        return highest
    except Exception as e:
        print(f"Error fetching chapters: {e}")
        return 0