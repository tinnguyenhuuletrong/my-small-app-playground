.PHONY: init run clean

init:
	uv venv
	uv sync

run:
	uv run streamlit run app.py

clean:
	rm -rf .venv