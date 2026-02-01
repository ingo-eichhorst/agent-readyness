/**
 * Simple Express-like application with TypeScript types.
 */

interface User {
  id: number;
  name: string;
  email: string;
  age?: number;
}

interface ApiResponse<T> {
  data: T;
  status: number;
  message: string;
}

function createUser(name: string, email: string, age?: number): User {
  if (!name) {
    throw new Error("name is required");
  }
  if (!email.includes("@")) {
    throw new Error("invalid email address");
  }
  return {
    id: Math.floor(Math.random() * 10000),
    name,
    email,
    age,
  };
}

function getGreeting(user: User): string {
  if (user.age !== undefined) {
    return `Hello, ${user.name} (age ${user.age})!`;
  }
  return `Hello, ${user.name}!`;
}

function listUsers(users: User[]): string[] {
  return users.map((u) => u.name);
}

export { User, ApiResponse, createUser, getGreeting, listUsers };
