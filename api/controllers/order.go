package controllers

import (
	"context"
	"elektron-canteen/api/controllers/utils"
	"elektron-canteen/api/data/addition"
	"elektron-canteen/api/data/meal"
	"elektron-canteen/api/data/menu"
	"elektron-canteen/api/data/order"
	"elektron-canteen/api/data/user"
	"errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrderController struct {
	order     order.Model
	menu      menu.Model
	user      user.Model
	meal      meal.Model
	addition  addition.Model
	validator *order.Validator
	oc        chan primitive.ObjectID
}

func NewOrderController() *OrderController {
	return &OrderController{
		order:     order.Instance(),
		menu:      menu.Instance(),
		user:      user.Instance(),
		meal:      meal.Instance(),
		addition:  addition.Instance(),
		validator: order.NewValidator(),
		oc:        make(chan primitive.ObjectID),
	}
}

func (c *OrderController) AddOrder(no order.NewOrder) (primitive.ObjectID, error) {
	ctx := context.Background()

	no.Status = order.WAITING
	no.Date = utils.UnixToFormattedDate(no.DueTime)

	// Validating new order
	if err := c.validator.ValidateOrder(no); err != nil {
		return primitive.ObjectID{}, err
	}

	if err := c.validator.ValidateUnixDate(no.DueTime); err != nil {
		return primitive.ObjectID{}, err
	}

	// Check if meal is in menu
	todayMenu, err := c.menu.QueryByDay(ctx, utils.UnixToFormattedDate(no.DueTime))
	if err != nil {
		return primitive.ObjectID{}, err
	}

	var isMealFound = false
	for _, am := range todayMenu.AvailableMeals {
		if no.Meal.Hex() == am {
			isMealFound = true
			break
		}
	}
	if !isMealFound {
		return primitive.ObjectID{}, errors.New("meal is not available")
	}

	// Query additions
	additions := []addition.Addition{}
	for _, addition := range no.Additions {
		id, err := primitive.ObjectIDFromHex(addition)
		if err != nil {
			return primitive.ObjectID{}, err
		}

		a, err := c.addition.QueryByID(ctx, id)
		if err != nil {
			return primitive.ObjectID{}, err
		}

		additions = append(additions, *a)
	}

	// Increase order number
	allOrders, err := c.order.QueryByDate(ctx, utils.UnixToFormattedDate(no.DueTime))
	if err != nil {
		return primitive.ObjectID{}, err
	}
	ordersAmount := len(allOrders)
	no.Number = ordersAmount + 1

	// Check if meal contains additions
	meal, err := c.meal.QueryByID(ctx, no.Meal)
	if err != nil {
		return primitive.ObjectID{}, err
	}

	//for _, addition := range meal.Additions {
	//	var isAdditionFound = false
	//	for _, orderAddition := range no.Additions {
	//		if orderAddition == addition {
	//			isAdditionFound = true
	//			break
	//		}
	//	}
	//
	//	if !isAdditionFound {
	//		return primitive.ObjectID{}, errors.New("meal doesn't contains that addition")
	//	}
	//}

	// Calculate needed points amount
	user, err := c.user.QueryByID(ctx, no.User)
	if err != nil {
		return primitive.ObjectID{}, err
	}

	var totalPrice float32
	for _, a := range additions {
		totalPrice += a.Price
	}
	totalPrice += meal.Price
	if user.Points-totalPrice < 0 {
		return primitive.ObjectID{}, errors.New("user has not enough points")
	}
	no.Price = totalPrice

	newPoints := user.Points - totalPrice
	err = c.user.UpdatePoints(ctx, user.ID, newPoints)
	if err != nil {
		return primitive.ObjectID{}, err
	}

	res, err := c.order.Create(ctx, no)
	if err != nil {
		return primitive.ObjectID{}, err
	}

	c.oc <- res

	return res, nil
}

func (c *OrderController) UpdateOrderStatus(orderID primitive.ObjectID, status string) error {
	ctx := context.Background()

	if status != order.WAITING && status != order.CANCELED && status != order.ACCEPTED && status != order.DECLINED && status != order.DONE {
		return errors.New("Wrong order status")
	}

	o, err := c.order.QueryByID(ctx, orderID)
	if err != nil {
		return err
	}

	if o.Status == order.CANCELED {
		return errors.New("can't change order status, order is cancelled")
	}

	return c.order.UpdateStatus(ctx, orderID, status)
}

func (c *OrderController) GetAllOrders() ([]order.Order, error) {
	ctx := context.Background()
	return c.order.QueryAll(ctx)
}

func (c *OrderController) GetOrdersByDate(date string) ([]order.Order, error) {
	ctx := context.Background()

	if err := c.validator.ValidateDate(date); err != nil {
		return nil, err
	}

	return c.order.QueryByDate(ctx, date)
}

func (c *OrderController) GetOrder(orderID primitive.ObjectID) (*order.Response, error) {
	ctx := context.Background()
	o, err := c.order.QueryByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	return o.ToResponse(ctx, c.meal, c.addition)
}

func (c *OrderController) CancelOrder(userID, orderID primitive.ObjectID) error {
	ctx := context.Background()
	userOrders, err := c.GetUserOrders(userID)

	if err != nil {
		return err
	}

	user, err := c.user.QueryByID(ctx, userID)
	if err != nil {
		return err
	}

	for _, uo := range userOrders {
		if uo.ID == orderID {
			if uo.Status == order.WAITING {
				err := c.order.UpdateStatus(ctx, uo.ID, order.CANCELED)
				if err != nil {
					return err
				}

				if uo.PaymentMethod == order.ONLINE_PAYMENT {
					newPoints := user.Points + uo.Price
					err := c.user.UpdatePoints(ctx, user.ID, newPoints)
					if err != nil {
						return err
					}
				}

				return nil
			} else {
				return errors.New("Cannot cancel order, order status: " + uo.Status)
			}
		}
	}

	return errors.New("order not found")
}

func (c *OrderController) GetUserOrders(userID primitive.ObjectID) ([]order.Order, error) {
	ctx := context.Background()
	return c.order.QueryByUser(ctx, userID)

}

func (c *OrderController) ListenForOrders(out chan *order.Response) {
	ctx := context.Background()

	for {
		orderID := <-c.oc
		o, err := c.order.QueryByID(ctx, orderID)
		if err != nil {
			out <- nil
		}

		or, err := o.ToResponse(ctx, c.meal, c.addition)
		if err != nil {
			out <- nil
		}

		out <- or
	}
}
