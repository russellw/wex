rd /s /q \t\workspace
md \t\workspace||exit /b
python run_engine.py --workspace \t\workspace --file test_prompts.txt|tee log.txt
dir \t\workspace|tee -a log.txt

