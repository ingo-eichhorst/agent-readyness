"""Simple web application with type annotations."""

from dataclasses import dataclass
from typing import Optional


@dataclass
class User:
    """A user in the system."""

    name: str
    email: str
    age: Optional[int] = None


def create_user(name: str, email: str, age: Optional[int] = None) -> User:
    """Create a new user with validation."""
    if not name:
        raise ValueError("name is required")
    if "@" not in email:
        raise ValueError("invalid email address")
    return User(name=name, email=email, age=age)


def get_greeting(user: User) -> str:
    """Return a greeting for the user."""
    if user.age is not None:
        return f"Hello, {user.name} (age {user.age})!"
    return f"Hello, {user.name}!"


def list_users(users: list[User]) -> list[str]:
    """Return a list of user names."""
    return [u.name for u in users]
