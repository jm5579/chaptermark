import sqlite3

DB_NAME = "manga_tracker.db"

def init_db():
    conn = sqlite3.connect(DB_NAME)
    c = conn.cursor()
    c.execute('''
        CREATE TABLE IF NOT EXISTS manga (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            title TEXT NOT NULL,
            alt_titles TEXT,
            mangadex_id TEXT,
            mangaupdates_id TEXT,
            anilist_id INTEGER,
            cover_url TEXT,
            description TEXT,
            status TEXT,
            genres TEXT,
            author TEXT,
            last_chapter_read REAL DEFAULT 0,
            reading_status TEXT DEFAULT 'Reading',
            personal_score REAL DEFAULT 0,
            mangadex_score REAL DEFAULT 0,
            mangaupdates_score REAL DEFAULT 0,
            anilist_score REAL DEFAULT 0,
            latest_chapter REAL DEFAULT 0,
            latest_chapter_mu REAL DEFAULT 0,
            latest_chapter_al REAL DEFAULT 0,
            added_on TEXT DEFAULT CURRENT_TIMESTAMP
        )
    ''')
    conn.commit()
    conn.close()

def get_all_manga():
    conn = sqlite3.connect(DB_NAME)
    c = conn.cursor()
    c.execute("SELECT * FROM manga ORDER BY title ASC")
    rows = c.fetchall()
    conn.close()
    return rows

def add_manga(data: dict):
    conn = sqlite3.connect(DB_NAME)
    c = conn.cursor()
    alt_titles = "|".join(data.get("alt_titles", []))
    c.execute('''
        INSERT INTO manga (
            title, alt_titles, mangadex_id, mangaupdates_id, anilist_id,
            cover_url, description, status, genres, author,
            mangadex_score, mangaupdates_score, anilist_score,
            latest_chapter, latest_chapter_mu, latest_chapter_al
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    ''', (
        data.get("title"), alt_titles,
        data.get("mangadex_id"), data.get("mangaupdates_id"),
        data.get("anilist_id"), data.get("cover_url"), data.get("description"),
        data.get("status"), data.get("genres"), data.get("author"),
        data.get("mangadex_score", 0), data.get("mangaupdates_score", 0),
        data.get("anilist_score", 0), data.get("latest_chapter", 0),
        data.get("latest_chapter_mu", 0), data.get("latest_chapter_al", 0)
    ))
    conn.commit()
    conn.close()

def update_manga(manga_id: int, last_chapter_read: float, reading_status: str, personal_score: float):
    conn = sqlite3.connect(DB_NAME)
    c = conn.cursor()
    c.execute('''
        UPDATE manga
        SET last_chapter_read = ?, reading_status = ?, personal_score = ?
        WHERE id = ?
    ''', (last_chapter_read, reading_status, personal_score, manga_id))
    conn.commit()
    conn.close()

def delete_manga(manga_id: int):
    conn = sqlite3.connect(DB_NAME)
    c = conn.cursor()
    c.execute("DELETE FROM manga WHERE id = ?", (manga_id,))
    conn.commit()
    conn.close()

def search_manga(query: str):
    conn = sqlite3.connect(DB_NAME)
    c = conn.cursor()
    c.execute(
        "SELECT * FROM manga WHERE title LIKE ? OR alt_titles LIKE ?",
        (f"%{query}%", f"%{query}%")
    )
    rows = c.fetchall()
    conn.close()
    return rows