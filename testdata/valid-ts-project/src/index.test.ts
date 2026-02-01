import { createUser, getGreeting, listUsers, User } from "./index";

describe("createUser", () => {
  it("should create a user with required fields", () => {
    const user = createUser("Alice", "alice@example.com");
    expect(user.name).toBe("Alice");
    expect(user.email).toBe("alice@example.com");
    expect(user.age).toBeUndefined();
  });

  it("should create a user with age", () => {
    const user = createUser("Bob", "bob@example.com", 30);
    expect(user.age).toBe(30);
  });

  it("should throw on invalid email", () => {
    expect(() => createUser("Alice", "not-an-email")).toThrow(
      "invalid email address"
    );
  });
});

describe("getGreeting", () => {
  it("should greet without age", () => {
    const user: User = { id: 1, name: "Alice", email: "alice@example.com" };
    expect(getGreeting(user)).toBe("Hello, Alice!");
  });

  it("should greet with age", () => {
    const user: User = {
      id: 2,
      name: "Bob",
      email: "bob@example.com",
      age: 25,
    };
    expect(getGreeting(user)).toBe("Hello, Bob (age 25)!");
  });
});
