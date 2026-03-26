# locustfile.py
import os
import random
import time
from locust import HttpUser, task, between
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class BoutiqueUser(HttpUser):
    """
    Эмулирует поведение пользователя в Google Boutique Shop
    """
    
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.session_id = None
        self.cart_items = []
        self.current_product = None
        self._users = int(os.environ.get('USERS', '10'))
        self._rate = int(os.environ.get('RATE', '10'))
        self._last_config_check = 0
        self._config_file = os.environ.get('CONFIG_FILE', '/config/config.env')
        self._check_interval = int(os.environ.get('REFRESH_INTERVAL', '5'))
        
    def on_start(self):
        """Инициализация пользователя при старте"""
        self.session_id = f"session_{random.randint(100000, 999999)}_{int(time.time())}"
        logger.info(f"User started - Session: {self.session_id}, Users: {self._users}, Rate: {self._rate}")
    
    def _check_config(self):
        """Проверка обновлений конфигурации"""
        try:
            if os.path.exists(self._config_file):
                mtime = os.path.getmtime(self._config_file)
                if mtime > self._last_config_check:
                    with open(self._config_file, 'r') as f:
                        for line in f:
                            if '=' in line:
                                key, value = line.strip().split('=', 1)
                                if key == 'RATE':
                                    new_rate = int(value)
                                    if new_rate != self._rate:
                                        logger.info(f"Rate changed: {self._rate} -> {new_rate}")
                                        self._rate = new_rate
                                elif key == 'USERS':
                                    new_users = int(value)
                                    if new_users != self._users:
                                        logger.info(f"Users changed: {self._users} -> {new_users}")
                                        self._users = new_users
                    self._last_config_check = mtime
        except Exception as e:
            logger.error(f"Error checking config: {e}")
    
    def wait_time(self):
        """Динамическая задержка между задачами"""
        self._check_config()
        
        # Рассчитываем задержку на основе целевого RPS
        if self._users > 0 and self._rate > 0:
            # RPS на одного пользователя = общий RPS / количество пользователей
            user_rate = self._rate / self._users
            # Задержка в секундах = 1 / RPS на пользователя
            avg_wait = 1.0 / user_rate if user_rate > 0 else 1.0
            # Добавляем случайность для реалистичности
            return random.uniform(avg_wait * 0.5, avg_wait * 1.5)
        return 1.0
    
    @task(3)
    def browse_products(self):
        """Просмотр списка продуктов"""
        with self.client.get("/", 
                             name="Browse Products",
                             catch_response=True,
                             headers={"Cookie": f"shop_session-id={self.session_id}"}) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Failed to browse products: {response.status_code}")
    
    @task(5)
    def view_product(self):
        """Просмотр деталей продукта"""
        products = [
            "OLJCESPC7Z", "66VCHSJNUP", "1YMWWN1N4O", "L9ECAV7KIM",
            "2ZYFJ3GM2N", "0PUK6V6EV0", "9SIQT8TOJO", "LS4PSXUNUM"
        ]
        product_id = random.choice(products)
        
        with self.client.get(f"/product/{product_id}",
                             name="View Product Details",
                             catch_response=True,
                             headers={"Cookie": f"shop_session-id={self.session_id}"}) as response:
            if response.status_code == 200:
                response.success()
                self.current_product = product_id
            else:
                response.failure(f"Failed to view product: {response.status_code}")
    
    @task(2)
    def add_to_cart(self):
        """Добавление товара в корзину"""
        if not self.current_product:
            self.current_product = "OLJCESPC7Z"
        
        data = {
            "product_id": self.current_product,
            "quantity": random.randint(1, 3)
        }
        
        with self.client.post("/cart",
                             json=data,
                             name="Add to Cart",
                             catch_response=True,
                             headers={"Cookie": f"shop_session-id={self.session_id}"}) as response:
            if response.status_code == 200:
                response.success()
                self.cart_items.append(self.current_product)
            else:
                response.failure(f"Failed to add to cart: {response.status_code}")
    
    @task(1)
    def view_cart(self):
        """Просмотр корзины"""
        with self.client.get("/cart",
                             name="View Cart",
                             catch_response=True,
                             headers={"Cookie": f"shop_session-id={self.session_id}"}) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Failed to view cart: {response.status_code}")
    
    @task(1)
    def checkout(self):
        """Оформление заказа"""
        if not self.cart_items:
            return
        
        checkout_data = {
            "email": f"user{random.randint(1, 1000)}@example.com",
            "street_address": f"{random.randint(100, 999)} Main St",
            "zip_code": f"{random.randint(10000, 99999)}",
            "city": "Springfield",
            "state": "IL",
            "country": "USA",
            "credit_card_number": "4432-8015-6152-0454",
            "credit_card_expiration_month": "01",
            "credit_card_expiration_year": "2030",
            "credit_card_cvv": "123"
        }
        
        with self.client.post("/checkout",
                             json=checkout_data,
                             name="Checkout",
                             catch_response=True,
                             headers={"Cookie": f"shop_session-id={self.session_id}"}) as response:
            if response.status_code == 200:
                response.success()
                self.cart_items = []
                logger.info(f"Checkout successful for {self.session_id}")
            else:
                response.failure(f"Failed to checkout: {response.status_code}")
    
    @task(2)
    def get_recommendations(self):
        """Получение рекомендаций"""
        product_id = random.choice(["OLJCESPC7Z", "66VCHSJNUP", "1YMWWN1N4O"])
        
        with self.client.get(f"/api/recommendations?product_id={product_id}",
                             name="Get Recommendations",
                             catch_response=True,
                             headers={"Cookie": f"shop_session-id={self.session_id}"}) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Failed to get recommendations: {response.status_code}")
