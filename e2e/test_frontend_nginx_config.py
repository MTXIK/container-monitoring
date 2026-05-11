from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parents[1]


class FrontendNginxConfigTest(unittest.TestCase):
    def test_frontend_nginx_serves_spa_routes(self):
        dockerfile = (ROOT / "frontend" / "Dockerfile").read_text(encoding="utf-8")
        nginx_config = ROOT / "frontend" / "nginx.conf"

        self.assertIn("nginx.conf", dockerfile)
        self.assertTrue(nginx_config.exists())
        self.assertIn("try_files $uri $uri/ /index.html;", nginx_config.read_text(encoding="utf-8"))
