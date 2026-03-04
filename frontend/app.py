import streamlit as st

from github_downloader import (
    GitfError,
    RateLimitError,
    collect_files,
    download_as_zip,
    human_size,
    parse_github_url,
    MAX_FILES,
    MAX_SIZE_WARN,
)

# ── Page config ──────────────────────────────────────────────────────────

st.set_page_config(
    page_title="Gitfetch",
    page_icon="📂",
    layout="centered",
)

# ── Theme CSS ────────────────────────────────────────────────────────────

_SHARED_CSS = """
[data-testid="stMetric"] {
    border-radius: 10px; padding: 12px 16px; text-align: center;
}
.file-row {
    padding: 4px 10px; font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', monospace;
    font-size: 13px; display: flex; justify-content: space-between;
}
.example-chip {
    display: inline-block; padding: 4px 12px; margin: 3px 4px;
    border-radius: 16px; font-size: 12px;
}
.empty-state {
    text-align: center; padding: 60px 20px;
    border: 2px dashed; border-radius: 12px; margin-top: 20px;
}
.empty-state .icon { font-size: 48px; margin-bottom: 12px; }
"""

DARK_CSS = f"""
<style>
{_SHARED_CSS}

/* ── Dark: backgrounds ── */
:root {{ --primary-color: #58a6ff !important; }}
html, body, [data-testid="stAppViewContainer"],
[data-testid="stAppViewContainer"] > section,
[data-testid="stMain"] {{
    background-color: #0d1117 !important;
}}
[data-testid="stSidebar"], [data-testid="stSidebar"] > div {{
    background-color: #161b22 !important;
}}
[data-testid="stHeader"] {{ background-color: #0d1117 !important; }}
header[data-testid="stHeader"] {{ background-color: #0d1117 !important; }}

/* ── Dark: text ── */
[data-testid="stAppViewContainer"] p,
[data-testid="stAppViewContainer"] span,
[data-testid="stAppViewContainer"] label,
[data-testid="stAppViewContainer"] h1,
[data-testid="stAppViewContainer"] h2,
[data-testid="stAppViewContainer"] h3,
[data-testid="stAppViewContainer"] div,
[data-testid="stSidebar"] p,
[data-testid="stSidebar"] span,
[data-testid="stSidebar"] label,
[data-testid="stSidebar"] h1,
[data-testid="stSidebar"] h2,
[data-testid="stSidebar"] h3,
[data-testid="stSidebar"] div {{
    color: #e6edf3 !important;
}}
[data-testid="stMarkdownContainer"] p,
[data-testid="stMarkdownContainer"] li,
[data-testid="stMarkdownContainer"] span {{
    color: #e6edf3 !important;
}}

/* ── Dark: inputs ── */
[data-testid="stTextInput"] input {{
    background-color: #161b22 !important; color: #e6edf3 !important;
    border-color: #30363d !important;
}}
[data-testid="stTextInput"] input::placeholder {{ color: #484f58 !important; }}
[data-testid="stTextInput"] label {{ color: #8b949e !important; }}

/* ── Dark: buttons ── */
[data-testid="stBaseButton-primary"] {{
    background-color: #238636 !important; color: #ffffff !important;
    border-color: #238636 !important;
}}
[data-testid="stBaseButton-primary"]:hover {{
    background-color: #2ea043 !important;
}}
[data-testid="stBaseButton-secondary"],
button[kind="secondary"] {{
    background-color: #21262d !important; color: #e6edf3 !important;
    border-color: #30363d !important;
}}
[data-testid="stBaseButton-secondary"]:hover,
button[kind="secondary"]:hover {{
    background-color: #30363d !important;
}}
[data-testid="stDownloadButton"] button {{
    background-color: #238636 !important; color: #ffffff !important;
    border-color: #238636 !important;
}}
[data-testid="stDownloadButton"] button:hover {{
    background-color: #2ea043 !important;
}}

/* ── Dark: metrics ── */
[data-testid="stMetric"] {{
    background: #161b22 !important; border: 1px solid #30363d !important;
}}
[data-testid="stMetricLabel"] p {{ color: #8b949e !important; }}
[data-testid="stMetricValue"] {{ color: #e6edf3 !important; }}

/* ── Dark: expander ── */
[data-testid="stExpander"] {{
    background: #161b22 !important; border: 1px solid #30363d !important; border-radius: 10px;
}}
[data-testid="stExpander"] summary span {{ color: #e6edf3 !important; }}
[data-testid="stExpander"] details div {{ color: #e6edf3 !important; }}

/* ── Dark: dividers ── */
[data-testid="stAppViewContainer"] hr {{ border-color: #21262d !important; }}
[data-testid="stSidebar"] hr {{ border-color: #30363d !important; }}

/* ── Dark: status widget ── */
[data-testid="stStatusWidget"] {{ background-color: #161b22 !important; border-color: #30363d !important; }}

/* ── Dark: progress bar ── */
[data-testid="stProgress"] > div > div {{ background-color: #21262d !important; }}
[role="progressbar"] > div {{ background-color: #58a6ff !important; }}

/* ── Dark: toggle track & thumb ── */
[data-testid="stCheckbox"] [data-baseweb="checkbox"] p {{
    color: #e6edf3 !important;
}}
[data-testid="stCheckbox"] [data-baseweb="checkbox"] > div:first-child {{
    background-color: #30363d !important;
}}
[data-testid="stCheckbox"] [data-baseweb="checkbox"]:has(input:checked) > div:first-child {{
    background-color: #58a6ff !important;
}}
[data-testid="stCheckbox"] [data-baseweb="checkbox"] > div:first-child > div {{
    background-color: #ffffff !important;
}}

/* ── Dark: sidebar inputs (token bar) ── */
[data-testid="stSidebar"] input,
[data-testid="stSidebar"] input[type="text"],
[data-testid="stSidebar"] input[type="password"] {{
    background-color: #0d1117 !important; color: #e6edf3 !important;
    border-color: #30363d !important; caret-color: #e6edf3 !important;
}}
[data-testid="stSidebar"] input::placeholder {{ color: #484f58 !important; }}
[data-testid="stSidebar"] [data-baseweb="input"] {{
    background-color: #0d1117 !important; border-color: #30363d !important;
}}
[data-testid="stSidebar"] [data-baseweb="base-input"] {{
    background-color: #0d1117 !important; border-color: #30363d !important;
}}
[data-testid="stSidebar"] [data-testid="stTextInput"] label {{
    color: #8b949e !important;
}}
[data-testid="stSidebar"] [data-testid="stTextInput"] button {{
    color: #8b949e !important; background: transparent !important;
}}
[data-testid="stSidebar"] [data-testid="stTextInput"] svg {{
    fill: #8b949e !important; color: #8b949e !important;
}}

/* ── Dark: custom classes ── */
.file-row {{ color: #e6edf3 !important; }}
.file-row:nth-child(even) {{ background: #161b22; }}
.file-row:nth-child(odd) {{ background: #0d1117; }}
.file-size {{ color: #8b949e !important; }}
.hero-sub {{ color: #8b949e !important; font-size: 1.1rem; margin-top: -8px; }}
.example-chip {{
    background: #21262d; border: 1px solid #30363d; color: #58a6ff;
}}
.empty-state {{ color: #484f58 !important; border-color: #30363d; }}

/* ── Dark: captions / small text ── */
[data-testid="stSidebar"] [data-testid="stCaptionContainer"] p {{ color: #484f58 !important; }}
small, .stCaption {{ color: #484f58 !important; }}
</style>
"""

LIGHT_CSS = f"""
<style>
{_SHARED_CSS}

/* ── Light: backgrounds ── */
:root {{ --primary-color: #0969da !important; }}
html, body, [data-testid="stAppViewContainer"],
[data-testid="stAppViewContainer"] > section,
[data-testid="stMain"] {{
    background-color: #ffffff !important;
}}
[data-testid="stSidebar"], [data-testid="stSidebar"] > div {{
    background-color: #f6f8fa !important;
}}
[data-testid="stHeader"] {{ background-color: #ffffff !important; }}
header[data-testid="stHeader"] {{ background-color: #ffffff !important; }}

/* ── Light: text ── */
[data-testid="stAppViewContainer"] p,
[data-testid="stAppViewContainer"] span,
[data-testid="stAppViewContainer"] label,
[data-testid="stAppViewContainer"] h1,
[data-testid="stAppViewContainer"] h2,
[data-testid="stAppViewContainer"] h3,
[data-testid="stAppViewContainer"] div,
[data-testid="stSidebar"] p,
[data-testid="stSidebar"] span,
[data-testid="stSidebar"] label,
[data-testid="stSidebar"] h1,
[data-testid="stSidebar"] h2,
[data-testid="stSidebar"] h3,
[data-testid="stSidebar"] div {{
    color: #1f2328 !important;
}}
[data-testid="stMarkdownContainer"] p,
[data-testid="stMarkdownContainer"] li,
[data-testid="stMarkdownContainer"] span {{
    color: #1f2328 !important;
}}

/* ── Light: inputs ── */
[data-testid="stTextInput"] input {{
    background-color: #ffffff !important; color: #1f2328 !important;
    border-color: #d0d7de !important;
}}
[data-testid="stTextInput"] input::placeholder {{ color: #8b949e !important; }}
[data-testid="stTextInput"] label {{ color: #656d76 !important; }}

/* ── Light: buttons ── */
[data-testid="stBaseButton-primary"] {{
    background-color: #0969da !important; color: #ffffff !important;
    border-color: #0969da !important;
}}
[data-testid="stBaseButton-primary"]:hover {{
    background-color: #0860ca !important;
}}
[data-testid="stBaseButton-secondary"],
button[kind="secondary"] {{
    background-color: #f6f8fa !important; color: #1f2328 !important;
    border-color: #d0d7de !important;
}}
[data-testid="stBaseButton-secondary"]:hover,
button[kind="secondary"]:hover {{
    background-color: #eaeef2 !important;
}}
[data-testid="stDownloadButton"] button {{
    background-color: #0969da !important; color: #ffffff !important;
    border-color: #0969da !important;
}}
[data-testid="stDownloadButton"] button:hover {{
    background-color: #0860ca !important;
}}

/* ── Light: metrics ── */
[data-testid="stMetric"] {{
    background: #f6f8fa !important; border: 1px solid #d0d7de !important;
}}
[data-testid="stMetricLabel"] p {{ color: #656d76 !important; }}
[data-testid="stMetricValue"] {{ color: #1f2328 !important; }}

/* ── Light: expander ── */
[data-testid="stExpander"] {{
    background: #f6f8fa !important; border: 1px solid #d0d7de !important; border-radius: 10px;
}}
[data-testid="stExpander"] summary span {{ color: #1f2328 !important; }}
[data-testid="stExpander"] details div {{ color: #1f2328 !important; }}

/* ── Light: dividers ── */
[data-testid="stAppViewContainer"] hr {{ border-color: #d8dee4 !important; }}
[data-testid="stSidebar"] hr {{ border-color: #d0d7de !important; }}

/* ── Light: status widget ── */
[data-testid="stStatusWidget"] {{ background-color: #f6f8fa !important; border-color: #d0d7de !important; }}

/* ── Light: progress bar ── */
[data-testid="stProgress"] > div > div {{ background-color: #eaeef2 !important; }}
[role="progressbar"] > div {{ background-color: #0969da !important; }}

/* ── Light: toggle track & thumb ── */
[data-testid="stCheckbox"] [data-baseweb="checkbox"] p {{
    color: #1f2328 !important;
}}
[data-testid="stCheckbox"] [data-baseweb="checkbox"] > div:first-child {{
    background-color: #d0d7de !important;
}}
[data-testid="stCheckbox"] [data-baseweb="checkbox"]:has(input:checked) > div:first-child {{
    background-color: #0969da !important;
}}
[data-testid="stCheckbox"] [data-baseweb="checkbox"] > div:first-child > div {{
    background-color: #ffffff !important;
}}

/* ── Light: sidebar inputs (token bar) ── */
[data-testid="stSidebar"] input,
[data-testid="stSidebar"] input[type="text"],
[data-testid="stSidebar"] input[type="password"] {{
    background-color: #ffffff !important; color: #1f2328 !important;
    border-color: #d0d7de !important; caret-color: #1f2328 !important;
}}
[data-testid="stSidebar"] input::placeholder {{ color: #8b949e !important; }}
[data-testid="stSidebar"] [data-baseweb="input"] {{
    background-color: #ffffff !important; border-color: #d0d7de !important;
}}
[data-testid="stSidebar"] [data-baseweb="base-input"] {{
    background-color: #ffffff !important; border-color: #d0d7de !important;
}}
[data-testid="stSidebar"] [data-testid="stTextInput"] label {{
    color: #656d76 !important;
}}
[data-testid="stSidebar"] [data-testid="stTextInput"] button {{
    color: #656d76 !important; background: transparent !important;
}}
[data-testid="stSidebar"] [data-testid="stTextInput"] svg {{
    fill: #656d76 !important; color: #656d76 !important;
}}

/* ── Light: custom classes ── */
.file-row {{ color: #1f2328 !important; }}
.file-row:nth-child(even) {{ background: #f6f8fa; }}
.file-row:nth-child(odd) {{ background: #ffffff; }}
.file-size {{ color: #656d76 !important; }}
.hero-sub {{ color: #656d76 !important; font-size: 1.1rem; margin-top: -8px; }}
.example-chip {{
    background: #f6f8fa; border: 1px solid #d0d7de; color: #0969da;
}}
.empty-state {{ color: #8b949e !important; border-color: #d0d7de; }}

/* ── Light: captions / small text ── */
[data-testid="stSidebar"] [data-testid="stCaptionContainer"] p {{ color: #8b949e !important; }}
small, .stCaption {{ color: #8b949e !important; }}
</style>
"""

# ── Session state defaults ───────────────────────────────────────────────

if "dark_mode" not in st.session_state:
    st.session_state.dark_mode = True
if "files" not in st.session_state:
    st.session_state.files = None
    st.session_state.info = None

# ── Inject active theme ─────────────────────────────────────────────────

st.markdown(DARK_CSS if st.session_state.dark_mode else LIGHT_CSS, unsafe_allow_html=True)

# ── Sidebar ──────────────────────────────────────────────────────────────

with st.sidebar:
    st.toggle("Dark mode", key="dark_mode")
    st.divider()

    st.header("Settings")
    gh_token = st.text_input(
        "GitHub Token (optional)",
        type="password",
        help="Increases API rate limit from 60 to 5,000 requests/hour.",
    )

    st.divider()

    with st.expander("How it works"):
        st.markdown(
            "1. Paste a GitHub folder URL\n"
            "2. **Fetch** scans the folder via the GitHub API\n"
            "3. **Download** grabs every file and bundles them into a ZIP"
        )

    st.caption(
        "Built with [Streamlit](https://streamlit.io)  \n"
        "CLI on [GitHub](https://github.com/chinmay706/Gitfetch)"
    )

# ── Header ───────────────────────────────────────────────────────────────

st.markdown("# :file_folder: Gitfetch")
st.markdown(
    '<p class="hero-sub">Download a specific folder from any public GitHub '
    "repository &mdash; without cloning the entire project.</p>",
    unsafe_allow_html=True,
)

# ── URL input ────────────────────────────────────────────────────────────

url_input = st.text_input(
    "GitHub folder URL",
    placeholder="https://github.com/owner/repo/tree/branch/path",
)

EXAMPLES = [
    ("Cobra docs", "https://github.com/spf13/cobra/tree/main/doc"),
    ("React DOM src", "https://github.com/facebook/react/tree/main/packages/react-dom/src"),
    ("VS Code extensions", "https://github.com/microsoft/vscode/tree/main/extensions"),
]
chips_html = " ".join(
    f'<span class="example-chip">{name}</span>' for name, _ in EXAMPLES
)
st.markdown(
    f"<p style='margin-bottom:2px;font-size:13px;'>Try an example:</p>{chips_html}",
    unsafe_allow_html=True,
)
example_cols = st.columns(len(EXAMPLES))
for i, (name, url) in enumerate(EXAMPLES):
    with example_cols[i]:
        if st.button(name, key=f"ex_{i}", use_container_width=True):
            st.session_state["url_autofill"] = url
            st.rerun()

if "url_autofill" in st.session_state:
    url_input = st.session_state.pop("url_autofill")

# ── Fetch files ──────────────────────────────────────────────────────────

col1, col2 = st.columns([1, 1])

with col1:
    fetch_clicked = st.button("Fetch file list", type="primary", use_container_width=True)

if fetch_clicked:
    if not url_input:
        st.error("Please enter a GitHub folder URL.")
        st.stop()

    try:
        info = parse_github_url(url_input)
    except GitfError as e:
        st.error(str(e))
        st.stop()

    st.session_state.info = info
    st.session_state.files = None

    status = st.status("Scanning repository...", expanded=True)
    try:
        files = collect_files(
            info,
            token=gh_token,
            on_status=lambda msg: status.update(label=msg),
        )
    except RateLimitError as e:
        status.update(label="Rate limited", state="error")
        st.error(str(e))
        st.stop()
    except GitfError as e:
        status.update(label="Error", state="error")
        st.error(str(e))
        st.stop()

    if not files:
        status.update(label="No files found", state="error")
        st.warning("No files found at the specified path.")
        st.stop()

    st.session_state.files = files
    status.update(label=f"Found {len(files)} files", state="complete")

# ── File preview & download ──────────────────────────────────────────────

info = st.session_state.info
files = st.session_state.files

if info and files:
    st.divider()

    cols = st.columns(4)
    cols[0].metric("Owner", info["owner"])
    cols[1].metric("Repo", info["repo"])
    cols[2].metric("Branch", info["branch"])
    cols[3].metric("Files", len(files))

    total_size = sum(f["size"] for f in files)

    if len(files) >= MAX_FILES:
        st.warning(
            f"File list capped at {MAX_FILES} files. "
            "The actual folder may contain more."
        )
    if total_size > MAX_SIZE_WARN:
        st.warning(
            f"Estimated total size is {human_size(total_size)}. "
            "Large downloads may be slow or fail on free hosting."
        )

    with st.expander(f"File list  ({len(files)} files, {human_size(total_size)} total)", expanded=False):
        prefix = info["path"] + "/" if info["path"] else ""
        rows_html = ""
        for f in files:
            rel = f["path"]
            if rel.startswith(prefix):
                rel = rel[len(prefix):]
            rows_html += (
                f'<div class="file-row">'
                f'<span>📄 {rel}</span>'
                f'<span class="file-size">{human_size(f["size"])}</span>'
                f'</div>'
            )
        st.markdown(rows_html, unsafe_allow_html=True)

    with col2:
        download_clicked = st.button("Download as ZIP", use_container_width=True)

    if download_clicked:
        base_prefix = info["path"]
        if base_prefix and not base_prefix.endswith("/"):
            base_prefix += "/"

        progress = st.progress(0, text="Downloading...")

        def update_progress(done, total):
            progress.progress(done / total, text=f"Downloading file {done}/{total}...")

        try:
            zip_buf = download_as_zip(
                files,
                base_prefix=base_prefix,
                token=gh_token,
                on_progress=update_progress,
            )
        except RateLimitError as e:
            progress.empty()
            st.error(str(e))
            st.stop()
        except GitfError as e:
            progress.empty()
            st.error(str(e))
            st.stop()

        progress.progress(1.0, text="Done!")

        folder_name = info["path"].rstrip("/").split("/")[-1] if info["path"] else info["repo"]

        st.download_button(
            label=f"Save {folder_name}.zip ({human_size(total_size)})",
            data=zip_buf,
            file_name=f"{folder_name}.zip",
            mime="application/zip",
            use_container_width=True,
        )

elif not fetch_clicked:
    st.markdown(
        '<div class="empty-state">'
        '<div class="icon">📂</div>'
        "<p>Paste a GitHub folder URL above and click <b>Fetch file list</b> to get started.</p>"
        "</div>",
        unsafe_allow_html=True,
    )
