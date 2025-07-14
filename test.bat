rd /s /q \t\workspace
md \t\workspace||exit /b
python run_engine.py --workspace \workspace %1|tee log.txt

