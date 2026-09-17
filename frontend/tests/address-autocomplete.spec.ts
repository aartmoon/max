import { expect, test } from "@playwright/test";

const house = {
  objectId: "2000",
  objectGuid: "20000000-0000-0000-0000-000000002000",
  parentObjectId: "1001",
  objectKind: "house",
  displayName: "д. 1",
  fullAddress: "г. Москва, ул. Тверская, д. 1",
};

const apartment = {
  objectId: "3000",
  objectGuid: "30000000-0000-0000-0000-000000003000",
  parentObjectId: "2000",
  objectKind: "apartment",
  displayName: "кв. 1",
  fullAddress: "г. Москва, ул. Тверская, д. 1, кв. 1",
};

test.beforeEach(async ({ page }) => {
  await page.route("https://st.max.ru/js/max-web-app.js", (route) => route.abort());
  await page.route("**/max/api/addresses/search*", async (route) => {
    const url = new URL(route.request().url());
    const kind = url.searchParams.get("kind");
    if (kind === "house") {
      await route.fulfill({ contentType: "application/json", body: JSON.stringify({ items: [house] }) });
      return;
    }
    expect(url.searchParams.get("parentObjectId")).toBe(house.objectId);
    await route.fulfill({ contentType: "application/json", body: JSON.stringify({ items: [apartment] }) });
  });
});

test("selects a GAR house and optional apartment and submits their IDs", async ({ page }) => {
  await page.goto("/max/requests/new");
  const houseInput = page.getByRole("combobox", { name: "Адрес дома" });
  await houseInput.fill("Тверская");
  await expect(page.getByRole("option", { name: house.fullAddress })).toBeVisible();
  await houseInput.press("ArrowDown");
  await houseInput.press("Enter");
  await expect(houseInput).toHaveValue(house.fullAddress);

  const apartmentInput = page.getByRole("combobox", { name: "Квартира" });
  await apartmentInput.focus();
  await expect(page.getByRole("option", { name: apartment.fullAddress })).toBeVisible();
  await page.getByRole("option", { name: apartment.fullAddress }).click();
  await page.getByLabel("Описание").fill("Не работает лифт");

  const posted = new Promise<string>((resolve) => {
    page.route("**/max/api/requests", async (route) => {
      resolve(route.request().postData() ?? "");
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ id: "7", address: apartment.fullAddress }),
      });
    });
  });
  await page.getByRole("button", { name: "Продолжить" }).click();
  const body = await posted;
  expect(body).toContain('name="houseObjectId"\r\n\r\n2000');
  expect(body).toContain('name="apartmentObjectId"\r\n\r\n3000');
});

test("editing a selected house clears stale IDs and blocks free text submission", async ({ page }) => {
  await page.goto("/max/requests/new");
  const houseInput = page.getByRole("combobox", { name: "Адрес дома" });
  await houseInput.fill("Тверская");
  await page.getByRole("option", { name: house.fullAddress }).click();
  await houseInput.fill("Произвольный адрес");
  await page.getByLabel("Описание").fill("Течёт труба");
  await page.getByRole("button", { name: "Продолжить" }).click();
  await expect(page.getByRole("alert")).toContainText("Выберите дом из списка");
});
