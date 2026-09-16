import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.route("https://st.max.ru/js/max-web-app.js", (route) =>
    route.abort(),
  );
});

test("application and client-side navigation stay under /max", async ({
  page,
}) => {
  await page.route("**/max/api/house", (route) =>
    route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        id: "1",
        address: "ул. Тестовая, д. 1",
        totalArea: 12480,
        livingArea: 9360,
        floors: 9,
        entrances: 4,
        apartments: 144,
        yearBuilt: 1987,
        organization: "УК «Тестовая»",
        manager: "Иван Иванов",
        contact: "+7 000 000-00-00",
      }),
    }),
  );
  await page.goto("/max/");

  await expect(
    page.getByRole("heading", { name: "Хороший дом начинается с вас." }),
  ).toBeVisible();

  await page.getByRole("link", { name: "Мой дом" }).first().click();
  await expect(page).toHaveURL(/\/max\/house$/);
  await expect(
    page.getByRole("heading", { name: "Мой дом", exact: true }),
  ).toBeVisible();
  await expect(page.getByText("ул. Тестовая, д. 1")).toBeVisible();
});
