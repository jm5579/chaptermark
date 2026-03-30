import streamlit as st
import requests
from PIL import Image
from io import BytesIO
from database import init_db, get_all_manga, add_manga, update_manga, delete_manga, search_manga
from mangadex import search_manga as mdex_search, get_latest_chapter
from mangaupdates import search_manga as mu_search
from anilist import search_manga as al_search
from dotenv import load_dotenv

load_dotenv()
init_db()

COLUMNS = ["id", "title", "alt_titles", "mangadex_id", "mangaupdates_id", "anilist_id",
           "cover_url", "description", "status", "genres", "author",
           "last_chapter_read", "reading_status", "personal_score",
           "mangadex_score", "mangaupdates_score", "anilist_score",
           "latest_chapter", "latest_chapter_mu", "latest_chapter_al", "added_on"]

def row_to_dict(row):
    return dict(zip(COLUMNS, row))

def load_image(url):
    try:
        response = requests.get(url, timeout=5)
        return Image.open(BytesIO(response.content))
    except:
        return None

st.set_page_config(page_title="Manga Tracker", layout="wide")
st.title("Manga Tracker")

tab1, tab2, tab3 = st.tabs(["My List", "Add Manga", "Table View"])

with tab2:
    st.subheader("Search and Add Manga")
    search_query = st.text_input("Search by title")

    if search_query:
        with st.spinner("Searching MangaDex..."):
            results = mdex_search(search_query)

        if not results:
            st.warning("No results found.")
        else:
            for i, manga in enumerate(results):
                with st.expander(manga["title"]):
                    col1, col2 = st.columns([1, 3])
                    with col1:
                        if manga.get("cover_url"):
                            img = load_image(manga["cover_url"])
                            if img:
                                st.image(img, width=120)
                    with col2:
                        st.write("Author:", manga.get("author", "Unknown"))
                        st.write("Status:", manga.get("status", "Unknown"))
                        st.write("Genres:", manga.get("genres", "Unknown"))
                        st.write("MangaDex Score:", manga.get("mangadex_score", 0))
                        if manga.get("alt_titles"):
                            st.write("Also known as:", ", ".join(manga["alt_titles"][:5]))
                        st.write(manga.get("description", "")[:300] + "...")

                    if st.button("Add to list", key=f"add_{i}"):
                        with st.spinner("Fetching data from all sources..."):
                            alt_titles = manga.get("alt_titles", [])
                            latest_mdex = get_latest_chapter(manga["mangadex_id"])
                            mu_id, mu_score, latest_mu = mu_search(manga["title"], alt_titles)
                            al_id, al_score, latest_al = al_search(manga["title"], alt_titles)
                            manga["latest_chapter"] = latest_mdex
                            manga["latest_chapter_mu"] = latest_mu
                            manga["latest_chapter_al"] = latest_al
                            manga["mangaupdates_id"] = mu_id
                            manga["mangaupdates_score"] = mu_score
                            manga["anilist_id"] = al_id
                            manga["anilist_score"] = al_score
                            add_manga(manga)
                        st.success(f"{manga['title']} added to your list!")

with tab1:
    st.subheader("My Reading List")

    col_search, col_status, col_sort = st.columns(3)
    with col_search:
        filter_query = st.text_input("Search your list", key="filter")
    with col_status:
        filter_status = st.selectbox("Filter by status", ["All", "Reading", "Paused", "Dropped", "Completed"])
    with col_sort:
        sort_by = st.selectbox("Sort by", ["Title", "Personal Score", "Latest Chapter", "Last Chapter Read"])

    rows = search_manga(filter_query) if filter_query else get_all_manga()
    manga_list = [row_to_dict(r) for r in rows]

    if filter_status != "All":
        manga_list = [m for m in manga_list if m["reading_status"] == filter_status]

    sort_map = {
        "Title": "title",
        "Personal Score": "personal_score",
        "Latest Chapter": "latest_chapter",
        "Last Chapter Read": "last_chapter_read"
    }
    manga_list.sort(key=lambda x: x[sort_map[sort_by]] or 0, reverse=(sort_by != "Title"))

    if not manga_list:
        st.info("No manga in your list yet. Go to Add Manga to get started.")
    else:
        cols = st.columns(4)
        for idx, manga in enumerate(manga_list):
            with cols[idx % 4]:
                if manga.get("cover_url"):
                    img = load_image(manga["cover_url"])
                    if img:
                        st.image(img, width=180)

                st.markdown(f"**{manga['title']}**")
                if manga.get("alt_titles"):
                    alts = manga["alt_titles"].split("|")[:2]
                    st.caption("aka: " + ", ".join(alts))
                st.caption(f"Status: {manga['reading_status']}")

                true_latest = max(
                    manga.get("latest_chapter") or 0,
                    manga.get("latest_chapter_mu") or 0,
                    manga.get("latest_chapter_al") or 0
                )
                chapters_behind = true_latest - (manga["last_chapter_read"] or 0)
                if chapters_behind > 0:
                    st.caption(f"Last read: Ch. {manga['last_chapter_read']} | Latest: Ch. {true_latest} | {int(chapters_behind)} behind")
                else:
                    st.caption(f"Last read: Ch. {manga['last_chapter_read']} | Up to date")

                st.caption(f"My score: {manga['personal_score']}/10")

                score_cols = st.columns(3)
                with score_cols[0]:
                    st.metric("MangaDex", manga["mangadex_score"] or "N/A")
                with score_cols[1]:
                    st.metric("MU", manga["mangaupdates_score"] or "N/A")
                with score_cols[2]:
                    st.metric("AL", manga["anilist_score"] or "N/A")

                with st.expander("Update"):
                    new_chapter = st.number_input(
                        "Last chapter read",
                        value=float(manga["last_chapter_read"] or 0),
                        step=0.5,
                        key=f"ch_{manga['id']}"
                    )
                    new_status = st.selectbox(
                        "Reading status",
                        ["Reading", "Paused", "Dropped", "Completed"],
                        index=["Reading", "Paused", "Dropped", "Completed"].index(manga["reading_status"]),
                        key=f"st_{manga['id']}"
                    )
                    new_score = st.slider(
                        "Personal score",
                        0.0, 10.0,
                        float(manga["personal_score"] or 0),
                        0.5,
                        key=f"sc_{manga['id']}"
                    )
                    if st.button("Save", key=f"save_{manga['id']}"):
                        update_manga(manga["id"], new_chapter, new_status, new_score)
                        st.success("Updated!")
                        st.rerun()
                    if st.button("Remove", key=f"del_{manga['id']}"):
                        delete_manga(manga["id"])
                        st.rerun()

with tab3:
    st.subheader("Table View")
    rows = get_all_manga()
    if not rows:
        st.info("No manga in your list yet.")
    else:
        import pandas as pd
        df = pd.DataFrame([row_to_dict(r) for r in rows])
        df = df[["title", "alt_titles", "reading_status", "last_chapter_read",
                 "latest_chapter", "latest_chapter_mu", "latest_chapter_al",
                 "personal_score", "mangadex_score", "mangaupdates_score",
                 "anilist_score", "genres", "author", "status"]]
        df.columns = ["Title", "Alt Titles", "Status", "Last Read",
                      "MangaDex Latest", "MU Latest", "AL Latest",
                      "My Score", "MangaDex Score", "MU Score", "AL Score",
                      "Genres", "Author", "Publication"]
        st.dataframe(df, use_container_width=True)