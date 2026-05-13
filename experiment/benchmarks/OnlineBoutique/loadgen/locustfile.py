# locustfile.py
import os
import random
import time
from locust import HttpUser, task, between, LoadTestShape, events
import logging
import math
from typing import List, Optional
import json

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class BoutiqueUser(HttpUser):
    """
    Эмулирует поведение пользователя в Google Boutique Shop
    """
    wait_time = between(1, 1.5)

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.session_id = None
        self.currency = "USD"
        self.cart_items = []
        self.current_product = None
        
    def on_start(self):
        """Инициализация пользователя при старте"""
        self.session_id = f"session_{random.randint(100000, 999999)}_{int(time.time())}"
        logger.info(f"User started - Session: {self.session_id}")
    
    def _get_headers(self):
        """Вспомогательный метод для передачи нужных куки"""
        return {
            "Cookie": f"shop_session-id={self.session_id}; shop_currency={self.currency}"
        }

    @task(3)
    def browse_products(self):
        """Просмотр списка продуктов"""
        with self.client.get("/", 
                             name="Browse Products",
                             catch_response=True,
                             headers=self._get_headers()) as response:
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
                             headers=self._get_headers()) as response:
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
                             data=data,
                             name="Add to Cart",
                             catch_response=True,
                             headers=self._get_headers()) as response:
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
                             headers=self._get_headers()) as response:
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
            "credit_card_number": "4432801561520454",
            "credit_card_expiration_month": "1",
            "credit_card_expiration_year": "2030",
            "credit_card_cvv": "123"
        }
        
        with self.client.post("/cart/checkout",
                             data=checkout_data,
                             name="Checkout",
                             catch_response=True,
                             headers=self._get_headers()) as response:
            if response.status_code == 200:
                response.success()
                self.cart_items = []
                logger.info(f"Checkout successful for {self.session_id}")
            else:
                response.failure(f"Failed to checkout: {response.status_code}")
    
    # @task(2)
    # def get_recommendations(self):
    #     """Получение рекомендаций"""
    #     product_id = random.choice(["OLJCESPC7Z", "66VCHSJNUP", "1YMWWN1N4O"])
        
    #     with self.client.get(f"/api/recommendations?product_id={product_id}",
    #                          name="Get Recommendations",
    #                          catch_response=True,
    #                          headers=self._get_headers()) as response:
    #         if response.status_code == 200:
    #             response.success()
    #         else:
    #             response.failure(f"Failed to get recommendations: {response.status_code}")

    @task(3)
    def set_currency(self):
        """Смена валюты"""
        currencies = ["USD", "EUR", "CAD", "JPY", "GBP", "AUD", "CHF"]
        currency = random.choice(currencies)
        
        with self.client.post("/setCurrency",
                            json={"currency_code": currency},
                            name="Set Currency",
                            catch_response=True,
                            headers=self._get_headers()) as response:
            if response.status_code == 200:
                response.success()
                self.user_currency = currency
            else:
                response.success()

class CustomVariableLoadShape(LoadTestShape):
    """
    Переменная нагрузка с плавным изменением по заданному массиву значений.
    
    Параметры:
    --load-array     Массив значений нагрузки (пользователи) через запятую
    --step-duration  Длительность одного шага в секундах (по умолчанию 60 - 1 минута)
    --smooth-steps   Количество промежуточных шагов для плавного перехода
                     Если 0 - изменения происходят резко, без промежуточных шагов
    """
    
    # Дефолтные значения
    load_array = []
    step_duration = 60  # секунд (1 минута)
    smooth_steps = 10   # количество промежуточных шагов для интерполяции
    
    def __init__(self):
        super().__init__()
        self._params_initialized = False
        self.load_array = self.__class__.load_array
        self.step_duration = self.__class__.step_duration
        self.smooth_steps = self.__class__.smooth_steps
        self._interpolated_load = []
        self._total_duration = 0
        self._instant_transitions = False  # Флаг для резких переходов
        
    def _init_params(self):
        """Безопасная инициализация параметров после появления runner"""
        if self._params_initialized:
            return
            
        if self.runner and self.runner.environment:
            parsed_options = self.runner.environment.parsed_options
            
            if parsed_options:
                # Получаем параметры из командной строки
                load_array_str = getattr(parsed_options, "load_array", "")
                if load_array_str:
                    try:
                        self.load_array = [int(x.strip()) for x in load_array_str.split(",")]
                    except ValueError:
                        print(f"Error parsing load_array: {load_array_str}")
                        self.load_array = []
                
                self.step_duration = getattr(parsed_options, "step_duration", self.step_duration)
                self.smooth_steps = getattr(parsed_options, "smooth_steps", self.smooth_steps)
        
        # Интерполируем нагрузку для плавного изменения
        self._interpolate_load()
        self._params_initialized = True
        
    def _interpolate_load(self):
        """Интерполяция значений для плавного или резкого изменения нагрузки"""
        if not self.load_array:
            self._interpolated_load = [(0, 0)]  # Нулевая нагрузка если массив пуст
            self._total_duration = 0
            return
        
        self._interpolated_load = []
        
        # Если smooth_steps = 0, используем резкие изменения
        if self.smooth_steps == 0:
            self._instant_transitions = True
            # Для резких переходов сохраняем только точки изменения
            for i, load_value in enumerate(self.load_array):
                time_offset = i * self.step_duration
                self._interpolated_load.append((time_offset, load_value))
            
            # Добавляем финальную точку для завершения
            end_time = len(self.load_array) * self.step_duration
            self._interpolated_load.append((end_time, 0))
            
            self._total_duration = end_time
            
            print(f"Load shape initialized (instant transitions):")
            print(f"  Load array: {self.load_array}")
            print(f"  Step duration: {self.step_duration}s")
            print(f"  Total duration: {self._total_duration}s")
            print(f"  Transition points: {len(self._interpolated_load)}")
            return
        
        # Блок для smooth_steps > 0 (плавные переходы)
        self._instant_transitions = False
        
        # Добавляем начальную точку
        if self.load_array[0] > 0:
            # Плавный старт от 0 до первого значения
            for i in range(self.smooth_steps):
                progress = i / self.smooth_steps
                users = int(self.load_array[0] * progress)
                time_offset = (i / self.smooth_steps) * self.step_duration
                self._interpolated_load.append((time_offset, users))
        
        # Интерполяция между точками
        for i in range(len(self.load_array) - 1):
            start_load = self.load_array[i]
            end_load = self.load_array[i + 1]
            start_time = i * self.step_duration
            
            for j in range(self.smooth_steps + 1):
                progress = j / self.smooth_steps
                
                # Используем косинусную интерполяцию для более плавного перехода
                smooth_progress = (1 - math.cos(progress * math.pi)) / 2
                
                users = int(start_load + (end_load - start_load) * smooth_progress)
                time_offset = start_time + (progress * self.step_duration)
                
                self._interpolated_load.append((time_offset, users))
        
        # Добавляем плавное завершение после последнего значения
        last_load = self.load_array[-1]
        if last_load > 0:
            for i in range(1, self.smooth_steps + 1):
                progress = i / self.smooth_steps
                users = int(last_load * (1 - progress))
                time_offset = (len(self.load_array) - 1) * self.step_duration + (progress * self.step_duration)
                self._interpolated_load.append((time_offset, users))
        
        # Удаляем дубликаты по времени (оставляем последнее значение)
        time_to_load = {}
        for time_offset, users in self._interpolated_load:
            time_to_load[time_offset] = users
        
        self._interpolated_load = sorted([(t, u) for t, u in time_to_load.items()])
        self._total_duration = len(self.load_array) * self.step_duration
        
        print(f"Load shape initialized (smooth transitions):")
        print(f"  Load array: {self.load_array}")
        print(f"  Step duration: {self.step_duration}s")
        print(f"  Total duration: {self._total_duration}s")
        print(f"  Interpolated points: {len(self._interpolated_load)}")
        
    def tick(self):
        # Инициализируем параметры при первом вызове tick
        self._init_params()
        
        run_time = self.get_run_time()
        
        # Получаем параметры из parsed_options
        spawn_rate = 10.0
        time_limit = None
        
        if self.runner and self.runner.environment and self.runner.environment.parsed_options:
            parsed = self.runner.environment.parsed_options
            
            if hasattr(parsed, 'spawn_rate') and parsed.spawn_rate:
                spawn_rate = float(parsed.spawn_rate)
            
            if hasattr(parsed, 'run_time') and parsed.run_time:
                time_limit = parsed.run_time
        
        # Проверка spawn_rate
        if not spawn_rate or spawn_rate <= 0:
            spawn_rate = 10.0
        
        # Проверка time_limit
        if time_limit and run_time > time_limit:
            return None
        
        # Если массив пустой или время вышло за пределы заданной нагрузки
        if not self.load_array or run_time >= self._total_duration:
            return (0, spawn_rate)
        
        # Находим текущее значение нагрузки
        users = self._get_current_load(run_time)
        
        return (users, spawn_rate)
    
    def _get_current_load(self, run_time):
        """Получение текущего значения нагрузки"""
        if not self._interpolated_load:
            return 0
        
        if self._instant_transitions:
            # Для резких переходов - используем ступенчатую функцию
            # Находим последнюю точку, которая не превышает текущее время
            current_load = 0
            for time_offset, load_value in self._interpolated_load:
                if run_time >= time_offset:
                    current_load = load_value
                else:
                    break
            return current_load
        else:
            # Для плавных переходов - линейная интерполяция
            # Если время меньше первой точки
            if run_time <= self._interpolated_load[0][0]:
                return self._interpolated_load[0][1]
            
            # Если время больше последней точки
            if run_time >= self._interpolated_load[-1][0]:
                return self._interpolated_load[-1][1]
            
            # Линейная интерполяция между двумя ближайшими точками
            for i in range(len(self._interpolated_load) - 1):
                t1, u1 = self._interpolated_load[i]
                t2, u2 = self._interpolated_load[i + 1]
                
                if t1 <= run_time <= t2:
                    # Линейная интерполяция
                    progress = (run_time - t1) / (t2 - t1)
                    return int(u1 + (u2 - u1) * progress)
            
            return 0

@events.init_command_line_parser.add_listener
def init_command_line_parser(parser):
    parser.add_argument("--load-array", type=str, default="",
                       help="Comma-separated array of user loads per minute (e.g., '10,50,100,50,10')")
    parser.add_argument("--step-duration", type=int, default=60,
                       help="Duration of each load step in seconds (default: 60)")
    parser.add_argument("--smooth-steps", type=int, default=20,
                       help="Number of interpolation steps for smooth transitions (default: 20)")
