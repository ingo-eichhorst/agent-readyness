"""Tests for the app module."""

import pytest
from app import User, create_user, get_greeting


def test_create_user():
    user = create_user("Alice", "alice@example.com")
    assert user.name == "Alice"
    assert user.email == "alice@example.com"
    assert user.age is None


def test_create_user_with_age():
    user = create_user("Bob", "bob@example.com", age=30)
    assert user.age == 30


def test_create_user_invalid_email():
    with pytest.raises(ValueError, match="invalid email"):
        create_user("Alice", "not-an-email")


def test_get_greeting():
    user = User(name="Alice", email="alice@example.com")
    assert get_greeting(user) == "Hello, Alice!"


def test_get_greeting_with_age():
    user = User(name="Bob", email="bob@example.com", age=25)
    assert get_greeting(user) == "Hello, Bob (age 25)!"


def test_create_user_empty_name():
    with pytest.raises(ValueError, match="name is required"):
        create_user("", "alice@example.com")
